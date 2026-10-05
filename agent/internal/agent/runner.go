package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/backup-saas/agent/internal/storage"
)

// connectionCredentials resolves DB credentials for a claimed job: a saved
// connection (per-claim, decrypted server-side) wins; otherwise the legacy
// agent environment config applies (backward compatible with pre-connection
// jobs and local dev). The returned clear function wipes the in-memory copy.
func pgForConn(conn *ConnectionConfig, fallback PgConfig) PgConfig {
	// A saved connection always wins when present — including no-auth
	// endpoints (empty user/password); per-field fallbacks below cover
	// fields the connection leaves blank.
	if conn != nil {
		port := conn.Port
		if port == 0 {
			port = 5432
		}
		host := conn.Host
		if host == "" {
			host = fallback.Host
		}
		user := conn.Username
		if user == "" {
			user = fallback.User
		}
		return PgConfig{Host: host, Port: port, User: user, Password: conn.Password}
	}
	return fallback
}

func pgFor(job *JobConfig, fallback PgConfig) PgConfig {
	if job == nil {
		return fallback
	}
	return pgForConn(job.Connection, fallback)
}

func mongoURIForConn(conn *ConnectionConfig, fallback string) string {
	// Saved connection wins when present; empty user means no-auth URI.
	if conn != nil {
		host := conn.Host
		if host == "" {
			host = "localhost"
		}
		port := conn.Port
		if port == 0 {
			port = 27017
		}
		return buildMongoURI(conn.Username, conn.Password, host, port)
	}
	return fallback
}

func mongoURIFor(job *JobConfig, fallback string) string {
	if job == nil {
		return fallback
	}
	return mongoURIForConn(job.Connection, fallback)
}

func mssqlForConn(conn *ConnectionConfig, fallback MssqlConfig) MssqlConfig {
	// Saved connection wins when present — including Windows integrated auth
	// (empty user/password); per-field fallbacks below cover blank fields.
	if conn != nil {
		host := conn.Host
		if host == "" {
			host = fallback.Server
		} else if conn.Port != 0 && conn.Port != 1433 {
			host = strings.Join([]string{host, itoa(conn.Port)}, ",")
		}
		user := conn.Username
		pw := conn.Password
		if user == "" {
			user = fallback.User
			pw = fallback.Password
		}
		return MssqlConfig{Server: host, User: user, Password: pw, BackupDir: fallback.BackupDir}
	}
	return fallback
}

func mssqlFor(job *JobConfig, fallback MssqlConfig) MssqlConfig {
	if job == nil {
		return fallback
	}
	return mssqlForConn(job.Connection, fallback)
}

func buildMongoURI(user, password, host string, port int) string {
	if user == "" {
		return strings.Join([]string{"mongodb://", host, ":", itoa(port), "/?authSource=admin"}, "")
	}
	return strings.Join([]string{"mongodb://", escapeUser(user), ":", escapeUser(password), "@", host, ":", itoa(port), "/?authSource=admin"}, "")
}

func escapeUser(s string) string {
	r := strings.ReplaceAll(s, "%", "%25")
	r = strings.ReplaceAll(r, "@", "%40")
	r = strings.ReplaceAll(r, "/", "%2F")
	r = strings.ReplaceAll(r, ":", "%3A")
	return r
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	pos := len(b)
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

func clearConnection(job *JobConfig) {
	if job.Connection != nil {
		// Best-effort memory hygiene: drop the secret as soon as the backup
		// payload is produced (upload uses storage creds, not DB creds).
		job.Connection.Password = ""
	}
}

func clearRestoreConnection(rs *Restore) {
	if rs.Connection != nil {
		rs.Connection.Password = ""
	}
}

// storageForJob resolves the provider for a claimed job. It prefers the
// self-contained Storage block new servers send; older servers only send
// the flat Storage* fields, which map to the same config.
func storageForJob(job *JobConfig) (storage.Storage, error) {
	st := job.Storage
	if strings.TrimSpace(st.Type) == "" {
		st = StorageConfig{
			Type:     job.StorageType,
			Path:     job.StoragePath,
			Bucket:   job.StorageBucket,
			Region:   job.StorageRegion,
			Endpoint: job.StorageEndpoint,
		}
	}
		var s3cfg *storage.S3Config
	if strings.EqualFold(strings.TrimSpace(st.Type), "S3") {
		s3cfg = &storage.S3Config{
			Bucket:       st.Bucket,
			Region:       st.Region,
			Endpoint:     st.Endpoint,
			UsePathStyle: st.UsePathStyle,
			AccessKey:    st.AccessKey,
			SecretKey:    st.SecretKey,
		}
	}
	return storage.ForJob(st.Type, st.Path, s3cfg)
}

// storageForRestore resolves the provider for a claimed restore, with the
// same old-server fallback as backups.
func storageForRestore(rs *Restore) (storage.Storage, error) {
	st := rs.Storage
	if strings.TrimSpace(st.Type) == "" {
		st = StorageConfig{
			Type:     rs.StorageType,
			Path:     "",
			Bucket:   "",
			Region:   "",
			Endpoint: "",
		}
	}
	// LOCAL restores read artifact files in place; the root only matters
	// for key-style paths, which legacy absolute paths bypass anyway.
	var root string
	var s3cfg *storage.S3Config
	if strings.EqualFold(strings.TrimSpace(st.Type), "S3") {
		s3cfg = &storage.S3Config{
			Bucket:       st.Bucket,
			Region:       st.Region,
			Endpoint:     st.Endpoint,
			UsePathStyle: st.UsePathStyle,
			AccessKey:    st.AccessKey,
			SecretKey:    st.SecretKey,
		}
	} else {
		root = st.Path
		if root == "" {
			// Restores address artifacts by their registered absolute path;
			// any existing directory root works for validation purposes.
			root = os.TempDir()
		}
	}
	return storage.ForJob(st.Type, root, s3cfg)
}

// downloadRestoreObject fetches a remote object to a temp file after
// verifying the disk can hold it. Fails fast with INSUFFICIENT_DISK_SPACE
// instead of filling the filesystem mid-download.
func (r *Runner) downloadRestoreObject(ctx context.Context, runID string, store storage.Storage, key string) (string, error) {
	head, err := store.Head(ctx, key)
	if err != nil {
		return "", fmt.Errorf("HEAD %s: %w", key, err)
	}
	tmpDir := os.TempDir()
	if free, err := storage.FreeDiskSpace(tmpDir); err == nil {
		// Object + decrypted copy + restored output can briefly coexist.
		if head.Size > 0 && free < uint64(head.Size)*2 {
			return "", fmt.Errorf("INSUFFICIENT_DISK_SPACE: need ~%d bytes, %d available in %s", head.Size*2, free, tmpDir)
		}
	} else {
		log.Printf("restore %s: disk-space check unavailable (%v), proceeding", runID, err)
	}
	rc, _, err := store.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", key, err)
	}
	defer rc.Close()
	tmp, err := os.CreateTemp(tmpDir, "vg-restore-*.tmp")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmp, rc); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("download %s: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	r.wsLog(runID, fmt.Sprintf("downloaded %d bytes", head.Size))
	return tmp.Name(), nil
}

// Runner executes one piece of work at a time, polling the server.
type Runner struct {
	cfg     Config
	client  *Client
	ws      *WSClient
	revoked bool
}

// isRevokedError detects the server's revocation rejection. Only the exact
// revocation signal matches — every other error (network, 5xx, invalid
// token typos) keeps the retry loop alive.
func isRevokedError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "revoked")
}

// noteRevoked latches the revoked state and reports whether the caller
// should stop polling. Revocation is terminal: the credential will never
// work again, so retrying is pointless and log-spammy.
func (r *Runner) noteRevoked(err error) bool {
	if isRevokedError(err) {
		r.revoked = true
		return true
	}
	return false
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
// A revoked credential stops the loop (safe stopped state): the service
// shows Stopped, and re-enrollment needs a fresh token + credential reset.
// An in-flight backup is NOT killed: executeBackup runs to its next safe
// reporting point, fails its server acknowledgements, and then the loop
// exits without claiming new work.
func (r *Runner) Run(ctx context.Context) {
	log.Printf("agent polling %s every %s", r.cfg.Server, r.cfg.PollInterval)
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	// Immediate first poll, then on every tick.
	for {
		r.pollOnce(ctx)
		if r.revoked {
			log.Print("agent credential revoked by administrator — stopping poll loop (delete the stored credential and re-enroll to resume)")
			return
		}
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
	// Explicit heartbeat: the server derives ONLINE/OFFLINE from recency.
	// Revocation stops the loop (safe state); anything else just logs and
	// the work poll below retries on its own cadence (no log spam beyond
	// the normal poll interval).
	if err := r.client.Heartbeat(); err != nil {
		if r.noteRevoked(err) {
			log.Printf("heartbeat rejected: %v", err)
			return
		}
		log.Printf("heartbeat: %v", err)
	}
	if r.pollBackups(ctx) {
		return // did backup work; restores next tick
	}
	r.pollRestores(ctx)
}

func (r *Runner) pollBackups(ctx context.Context) bool {
	runs, err := r.client.PendingRuns()
	if err != nil {
		if r.noteRevoked(err) {
			return false
		}
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
			if r.noteRevoked(err) {
				return false
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

	// Apply a per-job timeout so a hung dump/upload never blocks the agent
	// indefinitely. Configurable via AGENT_JOB_TIMEOUT (default 6h).
	timeout := r.cfg.JobTimeout
	if timeout <= 0 {
		timeout = 6 * time.Hour
	}
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ctx = jobCtx

	if job.SourceType != "FILESYSTEM" && job.SourceType != "POSTGRES" && job.SourceType != "MONGODB" && !isMssqlSource(job.SourceType) {
		r.failRun( runID, fmt.Sprintf("source type %s is not supported by agent v%s (supported: FILESYSTEM, POSTGRES, MONGODB, MSSQL_SERVER)", job.SourceType, Version), nil)
		return
	}
	// Storage provider for this job. Unknown/misconfigured targets fail
	// here — before the backup runs, never after the payload is produced.
	store, err := storageForJob(job)
	if err != nil {
		r.failRun(runID, "storage initialization failed", err)
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
		pg := pgFor(job, r.cfg.PG)
		dumpPath, size, err := BackupPostgres(pg, job.SourceDatabase, strings.ToUpper(job.Mode) == "COMPRESSED")
		clearConnection(job)
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
		mongoURI := mongoURIFor(job, r.cfg.MONGO.URI)
		switch format {
		case "ARCHIVE":
			label = "mongodump"
			outPath, size, err = BackupMongo(ctx, mongoURI, job.SourceDatabase)
		case "JSON":
			label = "mongo json export"
			outPath, size, err = ExportMongo(ctx, mongoURI, job.SourceDatabase, "JSON", strings.ToUpper(job.Mode) == "COMPRESSED")
		case "CSV":
			label = "mongo csv export"
			outPath, size, err = ExportMongo(ctx, mongoURI, job.SourceDatabase, "CSV", strings.ToUpper(job.Mode) == "COMPRESSED")
		default:
			r.failRun( runID, fmt.Sprintf("unknown export_format %q (want ARCHIVE, JSON, or CSV)", job.ExportFormat), nil)
			return
		}
		if err != nil {
			r.failRun( runID, label+" failed", err)
			return
		}
		clearConnection(job)
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
		bakPath, size, err := BackupMssql(ctx, mssqlFor(job, r.cfg.MSSQL), job.SourceDatabase, strings.ToUpper(job.Mode) == "COMPRESSED")
		clearConnection(job)
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

	// Upload through the storage provider. The object key (never a URL)
	// is what gets registered as the artifact's storage_path.
	key := storage.BuildKey(job.OrganizationID, job.JobID, runID, artifactName)
	payload, err := os.Open(archivePath)
	if err != nil {
		r.failRun(runID, "open payload failed", err)
		return
	}
	payloadStat, err := payload.Stat()
	if err != nil {
		payload.Close()
		r.failRun(runID, "stat payload failed", err)
		return
	}
	info, err := store.Put(ctx, key, payload, payloadStat.Size(), storage.PutOptions{
		ContentType: storage.ContentTypeForName(artifactName),
		Metadata: map[string]string{
			storage.MetaChecksumSHA256: archiveChecksum,
			storage.MetaRunID:          runID,
			storage.MetaArtifactName:   artifactName,
		},
	})
	payload.Close()
	if err != nil {
		// Local temp payloads are removed by the deferred cleanups above;
		// the next scheduled run retries the backup from scratch.
		r.failRun(runID, "upload failed", err)
		return
	}
	uploaded := info.Size
	r.wsLog(runID, fmt.Sprintf("uploaded %d bytes -> %s", uploaded, info.Location))

	// Verify before trusting: HEAD the object and compare size and the
	// VaultGuard SHA-256 metadata (S3 ETags are NOT content hashes for
	// multipart uploads). A mismatch deletes the remote object so a later
	// verify/reconcile never trusts it.
	head, err := store.Head(ctx, key)
	if err != nil {
		r.failRun(runID, "verify failed: HEAD", err)
		return
	}
	if head.Size != uploaded {
		_ = store.Delete(ctx, key)
		r.failRun(runID, fmt.Sprintf("verify failed: size mismatch (wrote %d, stored %d)", uploaded, head.Size), nil)
		return
	}
	if head.ChecksumSHA256 != "" && head.ChecksumSHA256 != strings.ToLower(archiveChecksum) {
		_ = store.Delete(ctx, key)
		r.failRun(runID, "verify failed: checksum mismatch", nil)
		return
	}
	dest := info.Location

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
			// Remove the just-uploaded object so a cancelled run leaves
			// no orphan behind (provider-aware: S3 key or local file).
			_ = store.Delete(ctx, key)
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
		if r.noteRevoked(err) {
			return
		}
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
			if r.noteRevoked(err) {
				return
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
func (r *Runner) restoreMongoExport(ctx context.Context, mongoURI, target, archivePath, format string) error {
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
		return RestoreMongoJSON(ctx, mongoURI, target, sqlPath)
	}
	if err := RestoreMongoCSV(ctx, mongoURI, target, sqlPath); err != nil {
		return err
	}
	log.Printf("CSV restore complete: values restored as strings (use ARCHIVE or JSON for exact types)")
	return nil
}

func (r *Runner) executeRestore(ctx context.Context, rs *Restore) {
	log.Printf("restore %s: extracting to %q", rs.ID, rs.DestinationPath)
	r.wsLog(rs.ID, fmt.Sprintf("restore started -> %q", rs.DestinationPath))
	store, err := storageForRestore(rs)
	if err != nil {
		msg := fmt.Sprintf("storage initialization failed: %v", err)
		log.Printf("restore %s FAILED: %s", rs.ID, msg)
		_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
		return
	}
	if ctx.Err() != nil {
		_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", "agent shutting down")
		return
	}
	// Remote objects (S3) download to a temp file first: deterministic
	// checksum/decrypt troubleshooting, and a disk-space gate before a
	// single byte moves. LOCAL artifacts stream straight from disk.
	archiveSrc := rs.StoragePath
	if _, isLocal := store.(*storage.LocalStorage); !isLocal {
		tmp, err := r.downloadRestoreObject(ctx, rs.ID, store, rs.StoragePath)
		if err != nil {
			msg := fmt.Sprintf("download failed: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		defer os.Remove(tmp)
		archiveSrc = tmp
	}
	// Decrypt first when the backup was encrypted. The data key arrives with
	// the claim (unwrapped server-side); plaintext never touches the server.
	archivePath := archiveSrc
	if rs.DataKey != "" {
		key, err := base64.StdEncoding.DecodeString(rs.DataKey)
		if err != nil {
			msg := fmt.Sprintf("bad data key: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		decPath := archiveSrc + ".dec"
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
		if err := RestorePostgres(pgForConn(rs.Connection, r.cfg.PG), rs.TargetDatabase, archivePath); err != nil {
			clearRestoreConnection(rs)
			msg := fmt.Sprintf("pg_restore failed: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		clearRestoreConnection(rs)
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
		if err := RestoreMssql(ctx, mssqlForConn(rs.Connection, r.cfg.MSSQL), rs.TargetDatabase, archivePath); err != nil {
			clearRestoreConnection(rs)
			msg := fmt.Sprintf("RESTORE DATABASE failed: %v", err)
			log.Printf("restore %s FAILED: %s", rs.ID, msg)
			_ = r.client.UpdateRestoreStatus(rs.ID, "FAILED", msg)
			return
		}
		clearRestoreConnection(rs)
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
		mongoURI := mongoURIForConn(rs.Connection, r.cfg.MONGO.URI)
		format := strings.ToUpper(strings.TrimSpace(rs.ExportFormat))
		if format == "" {
			format = "ARCHIVE"
		}
		var rerr error
		var rlabel string
		switch format {
		case "ARCHIVE":
			rlabel = "mongorestore"
			rerr = RestoreMongo(ctx, mongoURI, rs.SourceDatabase, target, archivePath)
		case "JSON", "CSV":
			rlabel = "mongo " + strings.ToLower(format) + " restore"
			rerr = r.restoreMongoExport(ctx, mongoURI, target, archivePath, format)
		default:
			rerr = fmt.Errorf("unknown export_format %q", rs.ExportFormat)
		}
		clearRestoreConnection(rs)
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
