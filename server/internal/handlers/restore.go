package handlers

import (
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RestoreHandler struct {
	db     *gorm.DB
	grpc   CommandDispatcher
	encKey []byte
}

func NewRestoreHandler(db *gorm.DB, grpc CommandDispatcher, encKey []byte) *RestoreHandler {
	return &RestoreHandler{db: db, grpc: grpc, encKey: encKey}
}

type createRestoreRequest struct {
	BackupRunID     string `json:"backup_run_id" binding:"required"`
	AgentID         string `json:"agent_id" binding:"required"`
	DestinationPath string `json:"destination_path"`
	TargetDatabase  string `json:"target_database"`
	// Confirmed must be true; forces the caller to explicitly acknowledge
	// that this operation will overwrite data at the destination.
	Confirmed bool `json:"confirmed" binding:"required"`
}

func (h *RestoreHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var jobs []models.RestoreJob
	h.db.Preload("BackupRun").Where("organization_id = ?", orgID).Order("created_at DESC").Find(&jobs)
	c.JSON(http.StatusOK, asArray(jobs))
}

func (h *RestoreHandler) Create(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req createRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Confirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirmed must be true to proceed with restore"})
		return
	}

	runID, err := uuid.Parse(req.BackupRunID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup_run_id"})
		return
	}
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	var run models.BackupRun
	if err := h.db.Where("id = ? AND organization_id = ?", runID, orgID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup run not found"})
		return
	}

	job := models.RestoreJob{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgID,
		BackupRunID:     runID,
		AgentID:         agentID,
		Status:          models.RestorePending,
		DestinationPath: req.DestinationPath,
		TargetDatabase:  req.TargetDatabase,
	}
	h.db.Create(&job)
	publishEntity(orgID.String(), realtime.TypeRestores, "created", job.ID.String())

	// Dispatch to agent
	h.grpc.SendCommand(agentID.String(), &pb.ServerCommand{
		Command: &pb.ServerCommand_StartRestore{
			StartRestore: &pb.StartRestoreCommand{
				RestoreId:       job.ID.String(),
				BackupRunId:     runID.String(),
				DestinationPath: req.DestinationPath,
				TargetDatabase:  req.TargetDatabase,
			},
		},
	})

	c.JSON(http.StatusCreated, job)
}

func (h *RestoreHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var job models.RestoreJob
	if err := h.db.Preload("BackupRun").
		Where("id = ? AND organization_id = ?", id, orgID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// UpdateStatus is called by the agent to update restore job progress.
// Auth is handled by AgentAuth middleware; agent is read from context.
func (h *RestoreHandler) UpdateStatus(c *gin.Context) {
	agent := c.MustGet("agent").(*models.Agent)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status       models.RestoreStatus `json:"status"`
		ErrorMessage string               `json:"error_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var job models.RestoreJob
	if err := h.db.Where("id = ? AND agent_id = ?", id, agent.ID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	updates := map[string]interface{}{"status": req.Status}
	if req.ErrorMessage != "" {
		updates["error_message"] = req.ErrorMessage
	}
	if req.Status == models.RestoreRunning && job.StartedAt == nil {
		now := time.Now()
		updates["started_at"] = &now
	}
	if req.Status == models.RestoreCompleted || req.Status == models.RestoreFailed {
		now := time.Now()
		updates["completed_at"] = &now
		if job.StartedAt != nil {
			updates["duration_seconds"] = int64(now.Sub(*job.StartedAt).Seconds())
		}
	}

	h.db.Model(&models.RestoreJob{}).
		Where("id = ? AND agent_id = ?", id, agent.ID).
		Updates(updates)
	publishEntity(agent.OrganizationID.String(), realtime.TypeRestores, "updated", id.String())

	c.JSON(http.StatusOK, gin.H{"updated": true})
}
