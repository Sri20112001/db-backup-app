package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/backup-saas/server/internal/models"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type BackupRunHandler struct {
	db     *gorm.DB
	grpc   CommandDispatcher
	encKey []byte
}

func NewBackupRunHandler(db *gorm.DB, grpc CommandDispatcher, encKey []byte) *BackupRunHandler {
	return &BackupRunHandler{db: db, grpc: grpc, encKey: encKey}
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

	query := h.db.Where("organization_id = ?", orgID).Order("created_at DESC")
	if jobID := c.Query("job_id"); jobID != "" {
		query = query.Where("backup_job_id = ?", jobID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.BackupRun{}).Count(&total)

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

// Cancel sends CANCEL_BACKUP to the agent and marks the run cancelled.
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

	h.db.Model(&run).Update("status", models.RunCancelled)

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

// authenticateAgent fetches the agent by ID (from X-Agent-ID header) then verifies
// the bearer token with bcrypt — O(1) DB lookup + one bcrypt compare.
func (h *BackupRunHandler) authenticateAgent(c *gin.Context) (*models.Agent, bool) {
	raw := extractBearerToken(c)
	if raw == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing agent token"})
		return nil, false
	}
	agentID, err := uuid.Parse(c.GetHeader("X-Agent-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid X-Agent-ID header"})
		return nil, false
	}
	var agent models.Agent
	if err := h.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(agent.TokenHash), []byte(raw)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return nil, false
	}
	return &agent, true
}

// UpdateStatus is called by the agent to update run progress/status.
func (h *BackupRunHandler) UpdateStatus(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}

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
			c.JSON(http.StatusOK, gin.H{"updated": true})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status transition: " + string(run.Status) + " → " + string(req.Status)})
		return
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

	h.db.Model(&run).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

// RegisterArtifact lets the agent record a backup artifact for a run.
func (h *BackupRunHandler) RegisterArtifact(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}

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
		Base:        models.Base{ID: uuid.New()},
		BackupRunID: runID,
		Name:        req.Name,
		Size:        req.Size,
		Checksum:    req.Checksum,
		StoragePath: req.StoragePath,
	}
	h.db.Create(&artifact)
	c.JSON(http.StatusCreated, artifact)
}

// RegisterChunk is idempotent: re-uploading chunk index N for the same artifact updates it.
func (h *BackupRunHandler) RegisterChunk(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}

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
