package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Runner executes one piece of work at a time, polling the server.
type Runner struct {
	cfg    Config
	client *Client
}

func NewRunner(cfg Config, client *Client) *Runner {
	return &Runner{cfg: cfg, client: client}
}

// Run loops until ctx is cancelled: poll for backup runs, then restores.
func (r *Runner) Run(ctx context.Context) {
	log.Printf("agent polling %s every %s", r.cfg.Server, r.cfg.PollInterval)
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	// Immediate first poll, then on every tick.
	for {
		r.pollOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Runner) pollOnce(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if r.pollBackups(ctx) {
		return // did backup work; restores next tick
	}
	r.pollRestores(ctx)
}

func (r *Runner) pollBackups(ctx context.Context) bool {
	runs, err := r.client.PendingRuns()
	if err != nil {
		log.Printf("poll runs: %v", err)
		return false
	}
	for _, run := range runs {
		if ctx.Err() != nil {
			return true
		}
		claim, err := r.client.ClaimRun(run.ID)
		if err != nil {
			if errors.Is(err, errClaimed) {
				continue
			}
			log.Printf("claim run %s: %v", run.ID, err)
			continue
		}
		r.executeBackup(ctx, claim.Run.ID, &claim.Config)
		return true
	}
	return false
}

func failRun(c *Client, runID, msg string, err error) {
	full := msg
	if err != nil {
		full = fmt.Sprintf("%s: %v", msg, err)
	}
	log.Printf("run %s FAILED: %s", runID, full)
	if uerr := c.UpdateRunStatus(runID, RunStatusUpdate{Status: "FAILED", ErrorMessage: full}); uerr != nil {
		log.Printf("report failure for run %s: %v", runID, uerr)
	}
}

func (r *Runner) executeBackup(ctx context.Context, runID string, job *JobConfig) {
	log.Printf("run %s: starting backup job %q (%s)", runID, job.Name, job.SourceType)

	if job.SourceType != "FILESYSTEM" && job.SourceType != "POSTGRES" {
		failRun(r.client, runID, fmt.Sprintf("source type %s is not supported by agent v%s (supported: FILESYSTEM, POSTGRES)", job.SourceType, Version), nil)
		return
	}
	if job.StorageType != "LOCAL" {
		failRun(r.client, runID, fmt.Sprintf("storage type %s is not supported by agent v%s (supported: LOCAL)", job.StorageType, Version), nil)
		return
	}
	if ctx.Err() != nil {
		failRun(r.client, runID, "agent shutting down", nil)
		return
	}

	lastPush := time.Now()
	progress := func(read, _ int64) {
		if time.Since(lastPush) < 5*time.Second {
			return
		}
		lastPush = time.Now()
		// Progress doubles as heartbeat (keeps the agent ONLINE).
		_ = r.client.UpdateRunStatus(runID, RunStatusUpdate{Status: "RUNNING", BytesRead: read})
	}

	// Produce the payload to store: a tar(.gz) archive for FILESYSTEM, a
	// pg_dump custom-format file for POSTGRES (already compressed).
	var payloadPath string
	var bytesRead, bytesCompressed int64
	var payloadChecksum string
	if job.SourceType == "FILESYSTEM" {
		res, err := BackupFilesystem(job.SourcePath, job.Mode, splitPatterns(job.IncludePatterns), splitPatterns(job.ExcludePatterns), progress)
		if err != nil {
			failRun(r.client, runID, "backup failed", err)
			return
		}
		defer os.Remove(res.ArchivePath)
		payloadPath, bytesRead, bytesCompressed = res.ArchivePath, res.BytesRead, res.BytesCompressed
		payloadChecksum = res.Checksum
	} else {
		if job.SourceDatabase == "" {
			failRun(r.client, runID, "POSTGRES job has no source_database configured", nil)
			return
		}
		dumpPath, size, err := BackupPostgres(r.cfg.PG, job.SourceDatabase, strings.ToUpper(job.Mode) == "COMPRESSED")
		if err != nil {
			failRun(r.client, runID, "pg_dump failed", err)
			return
		}
		defer os.Remove(dumpPath)
		checksum, err := ChecksumFile(dumpPath)
		if err != nil {
			failRun(r.client, runID, "checksum failed", err)
			return
		}
		payloadPath, bytesRead, bytesCompressed = dumpPath, size, size
		payloadChecksum = checksum
		log.Printf("run %s: pg_dump %q -> %d bytes", runID, job.SourceDatabase, size)
	}

	// Encrypt the finished archive when requested. Encryption happens here,
	// on the customer machine, before anything leaves for storage.
	// The per-backup data key is generated fresh and sent to the server
	// (envelope-encrypted at rest) with the completion report.
	archivePath := payloadPath
	archiveChecksum := payloadChecksum
	dataKeyB64 := ""
	artifactName := fmt.Sprintf("%s_%s", job.Name, runID)
	if job.Encrypted {
		key, err := GenerateDataKey()
		if err != nil {
			failRun(r.client, runID, "generate data key failed", err)
			return
		}
		encPath := payloadPath + ".enc"
		checksum, err := EncryptFile(key, payloadPath, encPath)
		if err != nil {
			os.Remove(encPath)
			failRun(r.client, runID, "encrypt failed", err)
			return
		}
		defer os.Remove(encPath)
		archivePath = encPath
		archiveChecksum = checksum
		dataKeyB64 = base64.StdEncoding.EncodeToString(key)
		artifactName += ".enc"
		log.Printf("run %s: encrypted archive with fresh data key", runID)
	}

	dest, uploaded, err := StoreLocal(job.StoragePath, job.Name, runID, archivePath)
	if err != nil {
		failRun(r.client, runID, "store failed", err)
		return
	}

	if _, err := r.client.RegisterArtifact(runID, artifactName, uploaded, archiveChecksum, dest); err != nil {
		log.Printf("run %s: register artifact: %v", runID, err)
	}

	// Walk the server state machine: RUNNING → UPLOADING → VERIFYING →
	// COMPLETED. The upload already happened locally above, so these are
	// quick successive transitions carrying the final counters.
	for _, status := range []string{"UPLOADING", "VERIFYING", "COMPLETED"} {
		err = r.client.UpdateRunStatus(runID, RunStatusUpdate{
			Status:          status,
			BytesRead:       bytesRead,
			BytesCompressed: bytesCompressed,
			BytesUploaded:   uploaded,
			StoragePath:     dest,
			Checksum:        archiveChecksum,
			DataKey:         dataKeyB64,
		})
		if err != nil {
			log.Printf("run %s: report %s: %v", runID, status, err)
			return
		}
	}
	log.Printf("run %s: COMPLETED %d bytes -> %s", runID, uploaded, dest)
}

func (r *Runner) pollRestores(ctx context.Context) {
	restores, err := r.client.PendingRestores()
	if err != nil {
		log.Printf("poll restores: %v", err)
		return
	}
	for _, rs := range restores {
		if ctx.Err() != nil {
			return
		}
		claim, err := r.client.ClaimRestore(rs.ID)
		if err != nil {
			if errors.Is(err, errClaimed) {
				continue
			}
			log.Printf("claim restore %s: %v", rs.ID, err)
			continue
		}
		r.executeRestore(ctx, claim)
		return
	}
}

func (r *Runner) executeRestore(ctx context.Context, rs *Restore) {
	log.Printf("restore %s: extracting to %q", rs.ID, rs.DestinationPath)
	if rs.StorageType != "" && rs.StorageType != "LOCAL" {
		msg := fmt.Sprintf("storage type %s is not supported by agent v%s (supported: LOCAL)", rs.StorageType, Version)
		log.Printf("restore %s FAILED: %s", rs.ID, msg)
		_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
		return
	}
	if ctx.Err() != nil {
		_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", "agent shutting down")
		return
	}
	// Decrypt first when the backup was encrypted. The data key arrives with
	// the claim (unwrapped server-side); plaintext never touches the server.
	archivePath := rs.StoragePath
	if rs.DataKey != "" {
		key, err := base64.StdEncoding.DecodeString(rs.DataKey)
		if err != nil {
			msg := fmt.Sprintf("bad data key: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		decPath := rs.StoragePath + ".dec"
		if err := DecryptFile(key, rs.StoragePath, decPath); err != nil {
			msg := fmt.Sprintf("decrypt failed: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		defer os.Remove(decPath)
		archivePath = decPath
		log.Printf("restore %s: decrypted archive", rs.ID)
	}
	if rs.SourceType == "POSTGRES" {
		if rs.TargetDatabase == "" {
			msg := "POSTGRES restore needs a target database"
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		log.Printf("restore %s: pg_restore to %q", rs.ID, rs.TargetDatabase)
		if err := RestorePostgres(r.cfg.PG, rs.TargetDatabase, archivePath); err != nil {
			msg := fmt.Sprintf("pg_restore failed: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		if err := r.client.UpdateRestoreStatus(rs.ID, "COMPLETED", ""); err != nil {
			log.Printf("restore %s: report completion: %v", rs.ID, err)
			return
		}
		log.Printf("restore %s: COMPLETED pg_restore to %s", rs.ID, rs.TargetDatabase)
		return
	}
	lastPush := time.Now()
	written, err := RestoreLocal(archivePath, rs.DestinationPath, func(read, _ int64) {
		if time.Since(lastPush) < 15*time.Second {
			return
		}
		lastPush = time.Now()
		_ = r.client.UpdateRestoreStatus(rs.ID, "RUNNING", "")
	})
	if err != nil {
		msg := fmt.Sprintf("restore failed: %v", err)
		log.Printf("restore %s FAILED: %s", rs.ID, msg)
		_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
		return
	}
	if err := r.client.UpdateRestoreStatus(rs.ID, "COMPLETED", ""); err != nil {
		log.Printf("restore %s: report completion: %v", rs.ID, err)
		return
	}
	log.Printf("restore %s: COMPLETED %d bytes -> %s", rs.ID, written, rs.DestinationPath)
}

// EnsureRegistered registers the agent on first run and caches credentials.
func EnsureRegistered(cfg Config) (*state, error) {
	if s, err := loadState(cfg.StateFile); err == nil {
		return s, nil
	}
	if cfg.RegistrationKey == "" {
		return nil, fmt.Errorf("no state file (%s) and AGENT_REGISTRATION_KEY is not set; generate a token in the UI first", cfg.StateFile)
	}
	c := NewClient(cfg.Server, "")
	out, err := c.Register(cfg.RegistrationKey, cfg.Hostname, Version)
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}
	s := &state{AgentID: out.AgentID, AgentToken: out.AgentToken}
	if err := saveState(cfg.StateFile, s); err != nil {
		return nil, fmt.Errorf("save state: %w", err)
	}
	log.Printf("registered as agent %s (credentials saved to %s)", out.AgentID, cfg.StateFile)
	return s, nil
}
