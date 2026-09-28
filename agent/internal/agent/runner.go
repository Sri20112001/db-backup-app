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
	ws     *WSClient
}

func NewRunner(cfg Config, client *Client, ws *WSClient) *Runner {
	return &Runner{cfg: cfg, client: client, ws: ws}
}

// wsLog streams one lifecycle line to dashboards watching runID (no-op
// without a socket).
func (r *Runner) wsLog(runID, line string) {
	if r.ws != nil {
		r.ws.Log(runID, line)
	}
}

// cancelledByUser merges the HTTP progress flag with instant socket cancels.
func (r *Runner) cancelledByUser(runID string, flag bool) bool {
	return flag || (r.ws != nil && r.ws.Cancelled(runID))
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
		if claim.CancelRequested {
			// Cancelled between listing and claim: nothing was produced.
			r.cancelRun( claim.Run.ID)
			return true
		}
		r.executeBackup(ctx, claim.Run.ID, &claim.Config)
		return true
	}
	return false
}

func (r *Runner) failRun(runID, msg string, err error) {
	full := msg
	if err != nil {
		full = fmt.Sprintf("%s: %v", msg, err)
	}
	log.Printf("run %s FAILED: %s", runID, full)
	r.wsLog(runID, "FAILED: "+full)
	if _, uerr := r.client.UpdateRunStatus(runID, RunStatusUpdate{Status: "FAILED", ErrorMessage: full}); uerr != nil {
		log.Printf("report failure for run %s: %v", runID, uerr)
	}
}

// cancelRun reports CANCELLED for a run the user cancelled mid-flight.
func (r *Runner) cancelRun(runID string) {
	log.Printf("run %s CANCELLED by user request", runID)
	r.wsLog(runID, "CANCELLED by user request")
	if r.ws != nil {
		r.ws.forget(runID)
	}
	if _, err := r.client.UpdateRunStatus(runID, RunStatusUpdate{Status: "CANCELLED"}); err != nil {
		log.Printf("report cancel for run %s: %v", runID, err)
	}
}

func (r *Runner) executeBackup(ctx context.Context, runID string, job *JobConfig) {
	log.Printf("run %s: starting backup job %q (%s)", runID, job.Name, job.SourceType)
	r.wsLog(runID, fmt.Sprintf("starting backup job %q (%s)", job.Name, job.SourceType))
	if r.cancelledByUser(runID, false) {
		r.cancelRun(runID)
		return
	}

	if job.SourceType != "FILESYSTEM" && job.SourceType != "POSTGRES" && job.SourceType != "MONGODB" && !isMssqlSource(job.SourceType) {
		r.failRun( runID, fmt.Sprintf("source type %s is not supported by agent v%s (supported: FILESYSTEM, POSTGRES, MONGODB, MSSQL_SERVER)", job.SourceType, Version), nil)
		return
	}
	if job.StorageType != "LOCAL" {
		r.failRun( runID, fmt.Sprintf("storage type %s is not supported by agent v%s (supported: LOCAL)", job.StorageType, Version), nil)
		return
	}
	if ctx.Err() != nil {
		r.failRun( runID, "agent shutting down", nil)
		return
	}

	lastPush := time.Now()
	cancelSeen := false
	progress := func(read, _ int64) {
		if time.Since(lastPush) < 5*time.Second {
			return
		}
		lastPush = time.Now()
		// Progress doubles as heartbeat (keeps the agent ONLINE) and as a
		// cancellation check: the server answers every report with the
		// current cancel_requested flag.
		if cancel, _ := r.client.UpdateRunStatus(runID, RunStatusUpdate{Status: "RUNNING", BytesRead: read}); r.cancelledByUser(runID, cancel) {
			cancelSeen = true
		}
	}

	// Produce the payload to store: a tar(.gz) archive for FILESYSTEM, a
	// plain-SQL dump for POSTGRES, a native .bak for MSSQL_SERVER.
	var payloadPath string
	var bytesRead, bytesCompressed int64
	var payloadChecksum string
	if job.SourceType == "FILESYSTEM" {
		res, err := BackupFilesystem(job.SourcePath, job.Mode, splitPatterns(job.IncludePatterns), splitPatterns(job.ExcludePatterns), progress)
		if err != nil {
			r.failRun( runID, "backup failed", err)
			return
		}
		defer os.Remove(res.ArchivePath)
		payloadPath, bytesRead, bytesCompressed = res.ArchivePath, res.BytesRead, res.BytesCompressed
		payloadChecksum = res.Checksum
	} else if job.SourceType == "POSTGRES" {
		if job.SourceDatabase == "" {
			r.failRun( runID, "POSTGRES job has no source_database configured", nil)
			return
		}
		dumpPath, size, err := BackupPostgres(r.cfg.PG, job.SourceDatabase, strings.ToUpper(job.Mode) == "COMPRESSED")
		if err != nil {
			r.failRun( runID, "pg_dump failed", err)
			return
		}
		defer os.Remove(dumpPath)
		checksum, err := ChecksumFile(dumpPath)
		if err != nil {
			r.failRun( runID, "checksum failed", err)
			return
		}
		payloadPath, bytesRead, bytesCompressed = dumpPath, size, size
		payloadChecksum = checksum
		log.Printf("run %s: pg_dump %q -> %d bytes", runID, job.SourceDatabase, size)
	} else if job.SourceType == "MONGODB" {
		if job.SourceDatabase == "" {
			r.failRun( runID, "MONGODB job has no source_database configured", nil)
			return
		}
		format := strings.ToUpper(strings.TrimSpace(job.ExportFormat))
		if format == "" {
			format = "ARCHIVE"
		}
		var outPath string
		var size int64
		var err error
		var label string
		switch format {
		case "ARCHIVE":
			label = "mongodump"
			outPath, size, err = BackupMongo(ctx, r.cfg.MONGO.URI, job.SourceDatabase)
		case "JSON":
			label = "mongo json export"
			outPath, size, err = ExportMongo(ctx, r.cfg.MONGO.URI, job.SourceDatabase, "JSON", strings.ToUpper(job.Mode) == "COMPRESSED")
		case "CSV":
			label = "mongo csv export"
			outPath, size, err = ExportMongo(ctx, r.cfg.MONGO.URI, job.SourceDatabase, "CSV", strings.ToUpper(job.Mode) == "COMPRESSED")
		default:
			r.failRun( runID, fmt.Sprintf("unknown export_format %q (want ARCHIVE, JSON, or CSV)", job.ExportFormat), nil)
			return
		}
		if err != nil {
			r.failRun( runID, label+" failed", err)
			return
		}
		defer os.Remove(outPath)
		checksum, err := ChecksumFile(outPath)
		if err != nil {
			r.failRun( runID, "checksum failed", err)
			return
		}
		payloadPath, bytesRead, bytesCompressed = outPath, size, size
		payloadChecksum = checksum
		log.Printf("run %s: %s %q -> %d bytes", runID, label, job.SourceDatabase, size)
		r.wsLog(runID, fmt.Sprintf("%s %q -> %d bytes", label, job.SourceDatabase, size))
	} else {
		if job.SourceDatabase == "" {
			r.failRun( runID, "MSSQL job has no source_database configured", nil)
			return
		}
		bakPath, size, err := BackupMssql(ctx, r.cfg.MSSQL, job.SourceDatabase, strings.ToUpper(job.Mode) == "COMPRESSED")
		if err != nil {
			r.failRun( runID, "BACKUP DATABASE failed", err)
			return
		}
		defer os.Remove(bakPath)
		checksum, err := ChecksumFile(bakPath)
		if err != nil {
			r.failRun( runID, "checksum failed", err)
			return
		}
		payloadPath, bytesRead, bytesCompressed = bakPath, size, size
		payloadChecksum = checksum
		log.Printf("run %s: BACKUP DATABASE %q -> %d bytes", runID, job.SourceDatabase, size)
	}

	// Post-backup data check: a zero-byte payload means the backup produced
	// nothing despite passing pre-checks — fail loudly instead of storing it.
	if size, err := payloadSize(payloadPath); err != nil || size == 0 {
		os.Remove(payloadPath)
		r.failRun( runID, "backup produced an empty payload (no data)", err)
		return
	}

	// Cooperative cancel: the user may have cancelled while we worked
	// (HTTP flag or instant socket command). Drop the payload and report
	// CANCELLED instead of storing it.
	if r.cancelledByUser(runID, cancelSeen) {
		os.Remove(payloadPath)
		r.cancelRun( runID)
		return
	}
	r.wsLog(runID, fmt.Sprintf("payload ready: %d bytes read", bytesRead))

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
			r.failRun( runID, "generate data key failed", err)
			return
		}
		encPath := payloadPath + ".enc"
		checksum, err := EncryptFile(key, payloadPath, encPath)
		if err != nil {
			os.Remove(encPath)
			r.failRun( runID, "encrypt failed", err)
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
		r.failRun( runID, "store failed", err)
		return
	}
	r.wsLog(runID, fmt.Sprintf("stored %d bytes -> %s", uploaded, dest))

	if _, err := r.client.RegisterArtifact(runID, artifactName, uploaded, archiveChecksum, dest); err != nil {
		log.Printf("run %s: register artifact: %v", runID, err)
	}

	// Walk the server state machine: RUNNING → UPLOADING → VERIFYING →
	// COMPLETED. The upload already happened locally above, so these are
	// quick successive transitions carrying the final counters.
	for _, status := range []string{"UPLOADING", "VERIFYING", "COMPLETED"} {
		cancel, err := r.client.UpdateRunStatus(runID, RunStatusUpdate{
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
		if r.cancelledByUser(runID, cancel) {
			os.Remove(dest)
			r.cancelRun( runID)
			return
		}
	}
	log.Printf("run %s: COMPLETED %d bytes -> %s", runID, uploaded, dest)
	r.wsLog(runID, fmt.Sprintf("COMPLETED %d bytes", uploaded))
	if r.ws != nil {
		r.ws.forget(runID)
	}
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

// restoreMongoExport replays a JSON/CSV export (gunzipping first when the
// stored file is compressed) into targetDB. CSV values land as strings —
// callers should surface that caveat, not fail on it.
func (r *Runner) restoreMongoExport(ctx context.Context, target, archivePath, format string) error {
	sqlPath := archivePath
	if strings.HasSuffix(archivePath, ".gz") {
		tmp, err := os.CreateTemp("", "vg-mongoimport-*.tmp")
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		tmpPath := tmp.Name()
		tmp.Close()
		defer os.Remove(tmpPath)
		if err := GunzipFile(archivePath, tmpPath); err != nil {
			return fmt.Errorf("gunzip export: %w", err)
		}
		sqlPath = tmpPath
	}
	if format == "JSON" {
		return RestoreMongoJSON(ctx, r.cfg.MONGO.URI, target, sqlPath)
	}
	if err := RestoreMongoCSV(ctx, r.cfg.MONGO.URI, target, sqlPath); err != nil {
		return err
	}
	log.Printf("CSV restore complete: values restored as strings (use ARCHIVE or JSON for exact types)")
	return nil
}

func (r *Runner) executeRestore(ctx context.Context, rs *Restore) {
	log.Printf("restore %s: extracting to %q", rs.ID, rs.DestinationPath)
	r.wsLog(rs.ID, fmt.Sprintf("restore started -> %q", rs.DestinationPath))
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
	if isMssqlSource(rs.SourceType) {
		if rs.TargetDatabase == "" {
			msg := "SQL Server restore needs a target database"
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		log.Printf("restore %s: RESTORE DATABASE to %q", rs.ID, rs.TargetDatabase)
		if err := RestoreMssql(ctx, r.cfg.MSSQL, rs.TargetDatabase, archivePath); err != nil {
			msg := fmt.Sprintf("RESTORE DATABASE failed: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		if err := r.client.UpdateRestoreStatus(rs.ID, "COMPLETED", ""); err != nil {
			log.Printf("restore %s: report completion: %v", rs.ID, err)
			return
		}
		log.Printf("restore %s: COMPLETED RESTORE DATABASE to %s", rs.ID, rs.TargetDatabase)
		return
	}
	if rs.SourceType == "MONGODB" {
		target := rs.TargetDatabase
		if target == "" {
			target = rs.SourceDatabase // same-name restore
		}
		if target == "" {
			msg := "MONGODB restore needs a target database"
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		log.Printf("restore %s: mongorestore %q -> %q", rs.ID, rs.SourceDatabase, target)
		format := strings.ToUpper(strings.TrimSpace(rs.ExportFormat))
		if format == "" {
			format = "ARCHIVE"
		}
		var rerr error
		var rlabel string
		switch format {
		case "ARCHIVE":
			rlabel = "mongorestore"
			rerr = RestoreMongo(ctx, r.cfg.MONGO.URI, rs.SourceDatabase, target, archivePath)
		case "JSON", "CSV":
			rlabel = "mongo " + strings.ToLower(format) + " restore"
			rerr = r.restoreMongoExport(ctx, target, archivePath, format)
		default:
			rerr = fmt.Errorf("unknown export_format %q", rs.ExportFormat)
		}
		if rerr != nil {
			msg := fmt.Sprintf("%s failed: %v", rlabel, rerr)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		if err := r.client.UpdateRestoreStatus(rs.ID, "COMPLETED", ""); err != nil {
			log.Printf("restore %s: report completion: %v", rs.ID, err)
			return
		}
		log.Printf("restore %s: COMPLETED mongorestore to %s", rs.ID, target)
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
		r.wsLog(rs.ID, "FAILED: "+msg)
		_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
		return
	}
	if err := r.client.UpdateRestoreStatus(rs.ID, "COMPLETED", ""); err != nil {
		log.Printf("restore %s: report completion: %v", rs.ID, err)
		return
	}
	log.Printf("restore %s: COMPLETED %d bytes -> %s", rs.ID, written, rs.DestinationPath)
	r.wsLog(rs.ID, fmt.Sprintf("COMPLETED %d bytes", written))
}

// EnsureRegistered registers the agent on first run and caches credentials.
func EnsureRegistered(cfg Config) (*state, error) {
	if s, err := loadState(cfg.StateFile); err == nil {
		return s, nil
	}
	if cfg.RegistrationKey == "" {
		return nil, fmt.Errorf("no state file (%s) and AGENT_REGISTRATION_KEY is not set; generate a token in the UI first", cfg.StateFile)
	}
	c := NewClientWithTLS(cfg.Server, "", cfg.TLSSkipVerify)
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
