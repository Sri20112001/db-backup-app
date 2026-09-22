package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/backup-saas/server/internal/models"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BackupRunHandler struct {
	db   *gorm.DB
	grpc CommandDispatcher
}

func NewBackupRunHandler(db *gorm.DB, grpc CommandDispatcher) *BackupRunHandler {
	return &BackupRunHandler{db: db, grpc: grpc}
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
		"data":  runs,
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
	if run.Status == models.RunCompleted || run.Status == models.RunFailed || run.Status == models.RunCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run is already in a terminal state"})
		return
	}

	h.db.Model(&run).Update("status", models.RunCancelled)

	h.grpc.SendCommand(run.AgentID.String(), &pb.ServerCommand{
		Command: &pb.ServerCommand_CancelBackup{
			CancelBackup: &pb.CancelBackupCommand{RunId: run.ID.String()},
		},
	})

	c.JSON(http.StatusOK, gin.H{"status": models.RunCancelled})
}

// UpdateStatus is called by the agent to update run progress/status.
// Authenticated via agent token in the Authorization header.
func (h *BackupRunHandler) UpdateStatus(c *gin.Context) {
	// Agent auth: Bearer <agent_token>
	agentToken := extractBearerToken(c)
	if agentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing agent token"})
		return
	}
	var agent models.Agent
	if err := h.db.Where("token = ?", agentToken).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	touchAgent(h.db, &agent)

	var req struct {
		Status          models.BackupRunStatus `json:"status"`
		BytesRead       int64                  `json:"bytes_read"`
		BytesCompressed int64                  `json:"bytes_compressed"`
		BytesUploaded   int64                  `json:"bytes_uploaded"`
		ErrorMessage    string                 `json:"error_message"`
		StoragePath     string                 `json:"storage_path"`
		Checksum        string                 `json:"checksum"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	}

	h.db.Model(&models.BackupRun{}).
		Where("id = ? AND agent_id = ?", runID, agent.ID).
		Updates(updates)

	c.JSON(http.StatusOK, gin.H{"updated": true})
}

// RegisterArtifact lets the agent record a backup artifact for a run.
func (h *BackupRunHandler) RegisterArtifact(c *gin.Context) {
	agentToken := extractBearerToken(c)
	var agent models.Agent
	if err := h.db.Where("token = ?", agentToken).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
		return
	}

	touchAgent(h.db, &agent)

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

// RegisterChunk lets the agent record an individual upload chunk.
func (h *BackupRunHandler) RegisterChunk(c *gin.Context) {
	agentToken := extractBearerToken(c)
	var agent models.Agent
	if err := h.db.Where("token = ?", agentToken).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	artifactID, err := uuid.Parse(c.Param("artifact_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artifact_id"})
		return
	}

	touchAgent(h.db, &agent)

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

	chunk := models.BackupChunk{
		Base:        models.Base{ID: uuid.New()},
		ArtifactID:  artifactID,
		Index:       req.Index,
		Size:        req.Size,
		Checksum:    req.Checksum,
		StoragePath: req.StoragePath,
		Uploaded:    req.Uploaded,
	}
	h.db.Create(&chunk)
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
	// Verify run belongs to org
	var run models.BackupRun
	if err := h.db.Where("id = ? AND organization_id = ?", runID, orgID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var artifacts []models.BackupArtifact
	h.db.Where("backup_run_id = ?", runID).Find(&artifacts)
	c.JSON(http.StatusOK, artifacts)
}

func extractBearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}
