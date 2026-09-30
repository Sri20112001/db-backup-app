package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/backup-saas/server/internal/services"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BackupRunHandler struct {
	db     *gorm.DB
	grpc   CommandDispatcher
	encKey []byte
	mailer *services.Mailer
}

func NewBackupRunHandler(db *gorm.DB, grpc CommandDispatcher, encKey []byte, mailer *services.Mailer) *BackupRunHandler {
	return &BackupRunHandler{db: db, grpc: grpc, encKey: encKey, mailer: mailer}
}

func (h *BackupRunHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	query := h.db.Where("backup_runs.organization_id = ?", orgID)
	if jobID := c.Query("job_id"); jobID != "" {
		query = query.Where("backup_runs.backup_job_id = ?", jobID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("backup_runs.status = ?", status)
	}
	// Parent job table: LEFT JOIN so search/sort by job name never drops
	// orphan rows, and preloaded job names stay consistent with ordering.
	query = query.Joins("LEFT JOIN backup_jobs ON backup_jobs.id = backup_runs.backup_job_id")
	// Free-text search across the parent job name.
	if q := strings.TrimSpace(c.Query("search")); q != "" {
		query = query.Where("backup_jobs.name ILIKE ?", "%"+escapeLike(q)+"%")
	}

	var total int64
	query.Model(&models.BackupRun{}).Count(&total)

	// Sorting: allowlisted columns only — raw input is never interpolated.
	sortCol := map[string]string{
		"job":      "backup_jobs.name",
		"started":  "backup_runs.started_at",
		"duration": "backup_runs.duration_seconds",
		"uploaded": "backup_runs.bytes_uploaded",
		"created":  "backup_runs.created_at",
	}[c.Query("sort")]
	if sortCol == "" {
		sortCol = "backup_runs.created_at"
	}
	order := "DESC"
	if strings.EqualFold(c.Query("order"), "asc") {
		order = "ASC"
	}
	// Secondary key keeps pagination stable for ties (e.g. equal durations).
	query = query.Order(sortCol + " " + order).Order("backup_runs.created_at DESC")

	var runs []models.BackupRun
	query.Preload("BackupJob").Limit(limit).Offset(offset).Find(&runs)

	c.JSON(http.StatusOK, gin.H{
		"data":  asArray(runs),
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *BackupRunHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var run models.BackupRun
	if err := h.db.Preload("BackupJob").
		Where("id = ? AND organization_id = ?", runID, orgID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, run)
}

// Cancel requests cancellation. PENDING runs (unclaimed) flip to CANCELLED
// immediately; claimed runs get cancel_requested=true, which the agent
// honors at stage boundaries and reports as CANCELLED itself.
func (h *BackupRunHandler) Cancel(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var run models.BackupRun
	if err := h.db.Where("id = ? AND organization_id = ?", runID, orgID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !models.CanTransition(run.Status, models.RunCancelled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot cancel run in status: " + string(run.Status)})
		return
	}

	updates := map[string]interface{}{"cancel_requested": true}
	if run.Status == models.RunPending {
		updates["status"] = models.RunCancelled
	}
	h.db.Model(&run).Updates(updates)

	if run.Status == models.RunPending {
		publishRun(run, models.RunCancelled, run.BytesRead, run.BytesCompressed, run.BytesUploaded, run.Checksum, run.ErrorMessage, run.StoragePath)
	} else {
		// Claimed run: nudge the live agent socket too so it aborts without
		// waiting for its next poll/progress round-trip.
		realtime.SendToAgent(realtime.DefaultHub, run.AgentID.String(), realtime.Event{
			Type:    realtime.TypeCancel,
			Payload: map[string]interface{}{"run_id": run.ID.String()},
		})
		publishRun(run, run.Status, run.BytesRead, run.BytesCompressed, run.BytesUploaded, run.Checksum, run.ErrorMessage, run.StoragePath)
	}

	h.grpc.SendCommand(run.AgentID.String(), &pb.ServerCommand{
		Command: &pb.ServerCommand_CancelBackup{
			CancelBackup: &pb.CancelBackupCommand{RunId: run.ID.String()},
		},
	})

	// Audit
	userID, _ := uuid.Parse(c.GetString("user_id"))
	h.db.Create(&models.AuditLog{
		OrganizationID: run.OrganizationID,
		UserID:         userID,
		Action:         "BACKUP_CANCELLED",
		Resource:       "backup_run",
		ResourceID:     run.ID.String(),
		IPAddress:      c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"status": models.RunCancelled})
}

// UpdateStatus is called by the agent to update run progress/status.
// Auth is handled by AgentAuth middleware; agent is read from context.
func (h *BackupRunHandler) UpdateStatus(c *gin.Context) {
	agent := c.MustGet("agent").(*models.Agent)

	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status          models.BackupRunStatus `json:"status"`
		BytesRead       int64                  `json:"bytes_read"`
		BytesCompressed int64                  `json:"bytes_compressed"`
		BytesUploaded   int64                  `json:"bytes_uploaded"`
		ErrorMessage    string                 `json:"error_message"`
		StoragePath     string                 `json:"storage_path"`
		Checksum        string                 `json:"checksum"`
		// DataKey is the base64 per-backup AES-256 data key, sent once by the
		// agent on completion of an encrypted backup. It is envelope-encrypted
		// with the server ENCRYPTION_KEY before storage.
		DataKey string `json:"data_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var run models.BackupRun
	if err := h.db.Where("id = ? AND agent_id = ?", runID, agent.ID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	if !models.CanTransition(run.Status, req.Status) {
		// Same-status progress reports (e.g. RUNNING → RUNNING heartbeats)
		// are idempotent: refresh counters without failing.
		if req.Status == run.Status {
			h.db.Model(&run).Updates(map[string]interface{}{
				"bytes_read":       req.BytesRead,
				"bytes_compressed": req.BytesCompressed,
				"bytes_uploaded":   req.BytesUploaded,
			})
			publishRun(run, req.Status, req.BytesRead, req.BytesCompressed, req.BytesUploaded, run.Checksum, run.ErrorMessage, run.StoragePath)
			c.JSON(http.StatusOK, gin.H{"updated": true})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status transition: " + string(run.Status) + " → " + string(req.Status)})
		return
	}

	// Validate reported storage_path is within the configured storage target directory.
	if req.StoragePath != "" {
		var target models.StorageTarget
		if err := h.db.Where("id = ?", run.StorageTargetID).First(&target).Error; err == nil && target.Path != "" {
			clean := filepath.Clean(req.StoragePath)
			base := filepath.Clean(target.Path)
			if !strings.HasPrefix(clean, base+string(filepath.Separator)) && clean != base {
				c.JSON(http.StatusBadRequest, gin.H{"error": "storage_path is outside the configured storage target directory"})
				return
			}
		}
	}

	updates := map[string]interface{}{
		"status":           req.Status,
		"bytes_read":       req.BytesRead,
		"bytes_compressed": req.BytesCompressed,
		"bytes_uploaded":   req.BytesUploaded,
		"error_message":    req.ErrorMessage,
		"storage_path":     req.StoragePath,
		"checksum":         req.Checksum,
	}
	if req.Status == models.RunCompleted || req.Status == models.RunFailed {
		now := time.Now()
		updates["completed_at"] = &now
		if run.StartedAt != nil {
			updates["duration_seconds"] = int64(now.Sub(*run.StartedAt).Seconds())
		}
	}
	if req.DataKey != "" {
		wrapped, err := encrypt(h.encKey, req.DataKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store data key"})
			return
		}
		updates["data_key_encrypted"] = wrapped
	}

	// Atomic state transition: only update if the current DB status still
	// matches what we read above, preventing a race between two concurrent
	// status reports from the same agent.
	res := h.db.Model(&models.BackupRun{}).
		Where("id = ? AND status = ?", run.ID, run.Status).
		Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "run status changed concurrently, please retry"})
		return
	}

	// Re-read: a Cancel may have landed concurrently with this report.
	var fresh models.BackupRun
	h.db.Select("cancel_requested").Where("id = ?", run.ID).First(&fresh)
	publishRun(run, req.Status, req.BytesRead, req.BytesCompressed, req.BytesUploaded, req.Checksum, req.ErrorMessage, req.StoragePath)

	// First FAILED report for a run raises an alert + email. Terminal states
	// never transition out, so exactly one FAILED report exists per run.
	if req.Status == models.RunFailed {
		var job models.BackupJob
		jobName := run.BackupJobID.String()
		if err := h.db.Where("id = ?", run.BackupJobID).First(&job).Error; err == nil {
			jobName = job.Name
		}
		// Classify the failure so the UI can show a structured error category.
		category := classifyFailure(req.ErrorMessage)
		h.db.Model(&models.BackupRun{}).Where("id = ?", run.ID).Update("failure_category", category)

		agentID := agent.ID
		jobID := run.BackupJobID
		alert := models.Alert{
			OrganizationID: run.OrganizationID,
			Type:           models.AlertBackupFailed,
			Title:          "Backup Failed: " + jobName,
			Message:        req.ErrorMessage,
			BackupJobID:    &jobID,
			AgentID:        &agentID,
		}
		h.db.Create(&alert)
		publishAlert(alert)
		services.NotifyOrg(h.db, h.mailer, run.OrganizationID,
			"[VaultGuard] Backup Failed: "+jobName,
			"Backup job "+jobName+" failed on agent "+agent.Name+":\n\n"+req.ErrorMessage)
	}

	c.JSON(http.StatusOK, gin.H{"updated": true, "cancel_requested": fresh.CancelRequested})
}

// Verify recomputes the SHA-256 of a COMPLETED run's stored file and
// compares it with the checksum the agent reported at upload time. Only
// LOCAL storage targets can be verified server-side (no S3/SMB client yet).
// A mismatch raises a VERIFY_FAILED alert + email; the run row is untouched.
func (h *BackupRunHandler) Verify(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var run models.BackupRun
	if err := h.db.Where("id = ? AND organization_id = ?", runID, orgID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if run.Status != models.RunCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only COMPLETED runs can be verified"})
		return
	}
	if run.Checksum == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run has no recorded checksum"})
		return
	}
	var target models.StorageTarget
	if err := h.db.Where("id = ?", run.StorageTargetID).First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storage target not found"})
		return
	}
	if target.Type != models.StorageLocal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification is only supported for LOCAL storage targets"})
		return
	}
	sum, err := sha256File(run.StoragePath)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not read stored file: " + err.Error()})
		return
	}
	if sum != strings.ToLower(run.Checksum) {
		jobID := run.BackupJobID
		alert := models.Alert{
			OrganizationID: run.OrganizationID,
			Type:           models.AlertVerifyFailed,
			Title:          "Backup verification failed",
			Message:        fmt.Sprintf("stored file hash mismatch for run %s (expected %s, got %s)", run.ID, run.Checksum, sum),
			BackupJobID:    &jobID,
			AgentID:        &run.AgentID,
		}
		h.db.Create(&alert)
		publishAlert(alert)
		services.NotifyOrg(h.db, h.mailer, run.OrganizationID,
			"[VaultGuard] Backup verification failed",
			fmt.Sprintf("Stored file for run %s does not match its recorded checksum.\nExpected: %s\nActual:   %s\n\nTreat this recovery point as suspect until investigated.",
				run.ID, run.Checksum, sum))
		c.JSON(http.StatusOK, gin.H{"match": false, "expected": run.Checksum, "actual": sum})
		return
	}
	c.JSON(http.StatusOK, gin.H{"match": true, "checksum": sum})
}

// sha256File streams a file into a hex digest without loading it.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RegisterArtifact lets the agent record a backup artifact for a run.
// Auth is handled by AgentAuth middleware; agent is read from context.
func (h *BackupRunHandler) RegisterArtifact(c *gin.Context) {
	agent := c.MustGet("agent").(*models.Agent)

	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
		return
	}

	// Verify run belongs to this agent
	var run models.BackupRun
	if err := h.db.Where("id = ? AND agent_id = ?", runID, agent.ID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Size        int64  `json:"size"`
		Checksum    string `json:"checksum"`
		StoragePath string `json:"storage_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	artifact := models.BackupArtifact{
		Base:            models.Base{ID: uuid.New()},
		BackupRunID:     runID,
		Name:            req.Name,
		Size:            req.Size,
		Checksum:        req.Checksum,
		StoragePath:     req.StoragePath,
		StorageTargetID: run.StorageTargetID,
	}
	h.db.Create(&artifact)
	c.JSON(http.StatusCreated, artifact)
}

// RegisterChunk is idempotent: re-uploading chunk index N for the same artifact updates it.
// Auth is handled by AgentAuth middleware; agent is read from context.
func (h *BackupRunHandler) RegisterChunk(c *gin.Context) {
	agent := c.MustGet("agent").(*models.Agent)

	artifactID, err := uuid.Parse(c.Param("artifact_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artifact_id"})
		return
	}

	// Verify artifact's run belongs to this agent
	var artifact models.BackupArtifact
	if err := h.db.First(&artifact, artifactID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
		return
	}
	var run models.BackupRun
	if err := h.db.Where("id = ? AND agent_id = ?", artifact.BackupRunID, agent.ID).First(&run).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		Index       int    `json:"index"`
		Size        int64  `json:"size"`
		Checksum    string `json:"checksum"`
		StoragePath string `json:"storage_path"`
		Uploaded    bool   `json:"uploaded"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Reject if same (artifact_id, index) exists with a different checksum — data integrity protection.
	var existing models.BackupChunk
	if err := h.db.Where("artifact_id = ? AND index = ?", artifactID, req.Index).First(&existing).Error; err == nil {
		if existing.Checksum != req.Checksum {
			c.JSON(http.StatusConflict, gin.H{"error": "chunk already exists with a different checksum"})
			return
		}
		// Same checksum — idempotent success, return existing.
		c.JSON(http.StatusOK, existing)
		return
	}

	chunk := models.BackupChunk{
		Base:        models.Base{ID: uuid.New()},
		ArtifactID:  artifactID,
		Index:       req.Index,
		Size:        req.Size,
		Checksum:    req.Checksum,
		StoragePath: req.StoragePath,
		Uploaded:    req.Uploaded,
	}
	if err := h.db.Create(&chunk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register chunk"})
		return
	}
	c.JSON(http.StatusCreated, chunk)
}

// ListArtifacts returns all artifacts for a backup run.
func (h *BackupRunHandler) ListArtifacts(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var run models.BackupRun
	if err := h.db.Where("id = ? AND organization_id = ?", runID, orgID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var artifacts []models.BackupArtifact
	h.db.Where("backup_run_id = ?", runID).Find(&artifacts)
	c.JSON(http.StatusOK, asArray(artifacts))
}

func extractBearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}

// classifyFailure maps a free-text error message to a structured category
// so the dashboard can display actionable error types instead of raw strings.
func classifyFailure(msg string) string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "authentication") || strings.Contains(m, "password") ||
		strings.Contains(m, "permission denied") || strings.Contains(m, "access denied") ||
		strings.Contains(m, "invalid credentials"):
		return "AUTH_ERROR"
	case strings.Contains(m, "no space left") || strings.Contains(m, "disk full") ||
		strings.Contains(m, "not enough space"):
		return "DISK_FULL"
	case strings.Contains(m, "connection refused") || strings.Contains(m, "no such host") ||
		strings.Contains(m, "network") || strings.Contains(m, "dial tcp") ||
		strings.Contains(m, "i/o timeout"):
		return "NETWORK_ERROR"
	case strings.Contains(m, "s3") || strings.Contains(m, "bucket") ||
		strings.Contains(m, "storage") || strings.Contains(m, "upload"):
		return "STORAGE_ERROR"
	case strings.Contains(m, "timeout") || strings.Contains(m, "deadline exceeded") ||
		strings.Contains(m, "context deadline"):
		return "TIMEOUT"
	case strings.Contains(m, "encrypt") || strings.Contains(m, "decrypt") ||
		strings.Contains(m, "data key"):
		return "ENCRYPTION_ERROR"
	case strings.Contains(m, "cancelled") || strings.Contains(m, "canceled"):
		return "CANCELLED"
	case strings.Contains(m, "source") || strings.Contains(m, "pg_dump") ||
		strings.Contains(m, "mongodump") || strings.Contains(m, "backup database"):
		return "SOURCE_ERROR"
	case strings.Contains(m, "verification") || strings.Contains(m, "checksum") ||
		strings.Contains(m, "hash mismatch"):
		return "VERIFICATION_ERROR"
	default:
		return "UNKNOWN"
	}
}
