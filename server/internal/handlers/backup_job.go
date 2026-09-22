package handlers

import (
	"net/http"

	"github.com/backup-saas/server/internal/models"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CommandDispatcher is satisfied by grpcserver.Server.
type CommandDispatcher interface {
	SendCommand(agentID string, cmd *pb.ServerCommand) bool
}

type BackupJobHandler struct {
	db   *gorm.DB
	grpc CommandDispatcher
}

func NewBackupJobHandler(db *gorm.DB, grpc CommandDispatcher) *BackupJobHandler {
	return &BackupJobHandler{db: db, grpc: grpc}
}

type createJobRequest struct {
	AgentID         string                  `json:"agent_id" binding:"required"`
	StorageTargetID string                  `json:"storage_target_id" binding:"required"`
	Name            string                  `json:"name" binding:"required"`
	SourceType      models.BackupSourceType `json:"source_type" binding:"required"`
	SourcePath      string                  `json:"source_path"`
	SourceDatabase  string                  `json:"source_database"`
	IncludePatterns string                  `json:"include_patterns"`
	ExcludePatterns string                  `json:"exclude_patterns"`
	Mode            models.BackupMode       `json:"mode"`
	Encrypted       bool                    `json:"encrypted"`
	RetentionDays   int                     `json:"retention_days"`
	CronExpr        string                  `json:"cron_expr"`
	Timezone        string                  `json:"timezone"`
}

func (h *BackupJobHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var jobs []models.BackupJob
	h.db.Preload("Schedule").Preload("Agent").Preload("StorageTarget").
		Where("organization_id = ?", orgID).Find(&jobs)
	c.JSON(http.StatusOK, jobs)
}

func (h *BackupJobHandler) Create(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}
	storageID, err := uuid.Parse(req.StorageTargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage_target_id"})
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = models.ModeNormal
	}
	retDays := req.RetentionDays
	if retDays == 0 {
		retDays = 30
	}

	job := models.BackupJob{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgID,
		AgentID:         agentID,
		StorageTargetID: storageID,
		Name:            req.Name,
		SourceType:      req.SourceType,
		SourcePath:      req.SourcePath,
		SourceDatabase:  req.SourceDatabase,
		IncludePatterns: req.IncludePatterns,
		ExcludePatterns: req.ExcludePatterns,
		Mode:            mode,
		Encrypted:       req.Encrypted,
		RetentionDays:   retDays,
		Enabled:         true,
	}
	if err := h.db.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}

	if req.CronExpr != "" {
		tz := req.Timezone
		if tz == "" {
			tz = "UTC"
		}
		h.db.Create(&models.BackupSchedule{
			BackupJobID: job.ID,
			CronExpr:    req.CronExpr,
			Timezone:    tz,
		})
	}

	h.db.Preload("Schedule").Preload("Agent").Preload("StorageTarget").First(&job, job.ID)
	c.JSON(http.StatusCreated, job)
}

func (h *BackupJobHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var job models.BackupJob
	if err := h.db.Preload("Schedule").Preload("Agent").Preload("StorageTarget").
		Where("id = ? AND organization_id = ?", jobID, orgID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *BackupJobHandler) Update(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var job models.BackupJob
	if err := h.db.Where("id = ? AND organization_id = ?", jobID, orgID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&job).Updates(map[string]interface{}{
		"name":             req.Name,
		"source_path":      req.SourcePath,
		"source_database":  req.SourceDatabase,
		"include_patterns": req.IncludePatterns,
		"exclude_patterns": req.ExcludePatterns,
		"mode":             req.Mode,
		"encrypted":        req.Encrypted,
		"retention_days":   req.RetentionDays,
	})

	if req.CronExpr != "" {
		tz := req.Timezone
		if tz == "" {
			tz = "UTC"
		}
		var sched models.BackupSchedule
		h.db.Where("backup_job_id = ?", job.ID).First(&sched)
		if sched.ID == uuid.Nil {
			h.db.Create(&models.BackupSchedule{BackupJobID: job.ID, CronExpr: req.CronExpr, Timezone: tz})
		} else {
			h.db.Model(&sched).Updates(map[string]interface{}{"cron_expr": req.CronExpr, "timezone": tz})
		}
	}

	h.db.Preload("Schedule").Preload("Agent").Preload("StorageTarget").First(&job, job.ID)
	c.JSON(http.StatusOK, job)
}

func (h *BackupJobHandler) Delete(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	h.db.Where("id = ? AND organization_id = ?", jobID, orgID).Delete(&models.BackupJob{})
	c.JSON(http.StatusNoContent, nil)
}

// RunNow creates a PENDING run and dispatches RUN_BACKUP to the agent via gRPC.
func (h *BackupJobHandler) RunNow(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var job models.BackupJob
	if err := h.db.Where("id = ? AND organization_id = ?", jobID, orgID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgID,
		BackupJobID:     job.ID,
		AgentID:         job.AgentID,
		StorageTargetID: job.StorageTargetID,
		Status:          models.RunPending,
		SourceType:      string(job.SourceType),
	}
	h.db.Create(&run)

	// Dispatch to agent if connected
	h.grpc.SendCommand(job.AgentID.String(), &pb.ServerCommand{
		Command: &pb.ServerCommand_RunBackup{
			RunBackup: &pb.RunBackupCommand{
				RunId: run.ID.String(),
				JobId: job.ID.String(),
			},
		},
	})

	c.JSON(http.StatusCreated, run)
}

func (h *BackupJobHandler) Enable(c *gin.Context) {
	h.setEnabled(c, true)
}

func (h *BackupJobHandler) Disable(c *gin.Context) {
	h.setEnabled(c, false)
}

func (h *BackupJobHandler) setEnabled(c *gin.Context, enabled bool) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	h.db.Model(&models.BackupJob{}).
		Where("id = ? AND organization_id = ?", jobID, orgID).
		Update("enabled", enabled)
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}
