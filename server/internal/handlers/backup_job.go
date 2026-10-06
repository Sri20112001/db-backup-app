package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
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
	AgentID          string                  `json:"agent_id" binding:"required"`
	StorageTargetID  string                  `json:"storage_target_id" binding:"required"`
	Name             string                  `json:"name" binding:"required"`
	SourceType       models.BackupSourceType `json:"source_type" binding:"required"`
	ConnectionID     string                  `json:"connection_id"`
	SourcePath       string                  `json:"source_path"`
	SourceDatabase   string                  `json:"source_database"`
	IncludePatterns  string                  `json:"include_patterns"`
	ExcludePatterns  string                  `json:"exclude_patterns"`
	Mode             models.BackupMode       `json:"mode"`
	Encrypted        bool                    `json:"encrypted"`
	RetentionDays    int                     `json:"retention_days"`
	// ExportFormat is only meaningful for MONGODB jobs; anything else is
	// rejected. Empty defaults to ARCHIVE.
	ExportFormat     string                  `json:"export_format"`
	CronExpr         string                  `json:"cron_expr"`
	Timezone         string                  `json:"timezone"`
	// PolicyID optionally attaches a BackupPolicy (how to back up). Empty =
	// job-inline settings (backward compatible, no migration needed).
	PolicyID         string                  `json:"policy_id"`
	SLATargetMinutes int                     `json:"sla_target_minutes"`
	RPOTargetMinutes int                     `json:"rpo_target_minutes"`
	RTOTargetMinutes int                     `json:"rto_target_minutes"`
}

// normalizeExportFormat validates the MongoDB export shape. Empty defaults
// to ARCHIVE; other source types must leave it empty.
func normalizeExportFormat(sourceType models.BackupSourceType, raw string) (string, error) {
	format := strings.ToUpper(strings.TrimSpace(raw))
	if format == "" {
		format = "ARCHIVE"
	}
	if sourceType == models.SourceMongo {
		switch format {
		case "ARCHIVE", "JSON", "CSV":
			return format, nil
		default:
			return "", fmt.Errorf("export_format must be ARCHIVE, JSON, or CSV")
		}
	}
	if raw != "" {
		return "", fmt.Errorf("export_format only applies to MONGODB jobs")
	}
	return format, nil
}

func (h *BackupJobHandler) List(c *gin.Context) {	orgID := c.MustGet("org_id").(uuid.UUID)
	var jobs []models.BackupJob
	h.db.Preload("Schedule").Preload("Policy").Preload("Agent").Preload("StorageTarget").Preload("Connection").
		Where("organization_id = ?", orgID).Find(&jobs)
	c.JSON(http.StatusOK, asArray(jobs))
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
	exportFormat, err := normalizeExportFormat(req.SourceType, req.ExportFormat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	retDays := req.RetentionDays
	if retDays == 0 {
		retDays = 30
	}

	// Resolve optional connection reference: must belong to this org, and
	// database jobs should pin the same agent as the connection so the
	// agent that holds network access runs the backup.
	var connID *uuid.UUID
	if strings.TrimSpace(req.ConnectionID) != "" {
		parsed, err := uuid.Parse(req.ConnectionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection_id"})
			return
		}
		var conn models.DatabaseConnection
		if err := h.db.Where("id = ? AND organization_id = ?", parsed, orgID).First(&conn).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "connection not found in this organization"})
			return
		}
		if isDatabaseSource(req.SourceType) && conn.AgentID != agentID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "connection belongs to a different agent; select the connection's agent"})
			return
		}
		connID = &parsed
	} else if isDatabaseSource(req.SourceType) && strings.TrimSpace(req.SourceDatabase) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_database is required for database jobs"})
		return
	}

	// Resolve optional policy reference: must belong to this org. Policies
	// are source-agnostic by design (how, not what), so no compat check.
	var policyID *uuid.UUID
	if strings.TrimSpace(req.PolicyID) != "" {
		parsed, err := uuid.Parse(req.PolicyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy_id"})
			return
		}
		var policy models.BackupPolicy
		if err := h.db.Where("id = ? AND organization_id = ?", parsed, orgID).First(&policy).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "policy not found in this organization"})
			return
		}
		policyID = &parsed
	}

	job := models.BackupJob{
		Base:             models.Base{ID: uuid.New()},
		OrganizationID:   orgID,
		AgentID:          agentID,
		StorageTargetID:  storageID,
		PolicyID:         policyID,
		Name:             req.Name,
		SourceType:       req.SourceType,
		ConnectionID:     connID,
		SourcePath:       req.SourcePath,
		SourceDatabase:   req.SourceDatabase,
		IncludePatterns:  req.IncludePatterns,
		ExcludePatterns:  req.ExcludePatterns,
		Mode:             mode,
		Encrypted:        req.Encrypted,
		RetentionDays:    retDays,
		ExportFormat:     exportFormat,
		Enabled:          true,
		SLATargetMinutes: req.SLATargetMinutes,
		RPOTargetMinutes: req.RPOTargetMinutes,
		RTOTargetMinutes: req.RTOTargetMinutes,
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

	h.db.Preload("Schedule").Preload("Policy").Preload("Agent").Preload("StorageTarget").Preload("Connection").First(&job, job.ID)
	publishEntity(orgID.String(), realtime.TypeJobs, "created", job.ID.String())
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
	if err := h.db.Preload("Schedule").Preload("Policy").Preload("Agent").Preload("StorageTarget").Preload("Connection").
		Where("id = ? AND organization_id = ?", jobID, orgID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func isDatabaseSource(t models.BackupSourceType) bool {
	return t == models.SourcePostgres || t == models.SourceMongo ||
		t == models.SourceSQLServer || t == models.SourceMssqlServer
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

	exportFormat, err := normalizeExportFormat(job.SourceType, req.ExportFormat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"name":               req.Name,
		"source_path":        req.SourcePath,
		"source_database":    req.SourceDatabase,
		"include_patterns":   req.IncludePatterns,
		"exclude_patterns":   req.ExcludePatterns,
		"mode":               req.Mode,
		"encrypted":          req.Encrypted,
		"retention_days":     req.RetentionDays,
		"export_format":      exportFormat,
		"sla_target_minutes": req.SLATargetMinutes,
		"rpo_target_minutes": req.RPOTargetMinutes,
		"rto_target_minutes": req.RTOTargetMinutes,
	}

	// Connection reassignment is optional; empty string leaves it unchanged,
	// explicit null clears it (filesystem migration). Frontend sends
	// connection_id only when the user picks one.
	if strings.TrimSpace(req.ConnectionID) != "" {
		parsed, err := uuid.Parse(req.ConnectionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection_id"})
			return
		}
		var conn models.DatabaseConnection
		if err := h.db.Where("id = ? AND organization_id = ?", parsed, orgID).First(&conn).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "connection not found in this organization"})
			return
		}
		updates["connection_id"] = parsed
	}

	// Policy attach/change; empty leaves it unchanged. Detach happens via
	// policy DELETE (which falls attached jobs back to inline settings).
	if strings.TrimSpace(req.PolicyID) != "" {
		parsed, err := uuid.Parse(req.PolicyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy_id"})
			return
		}
		var policy models.BackupPolicy
		if err := h.db.Where("id = ? AND organization_id = ?", parsed, orgID).First(&policy).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "policy not found in this organization"})
			return
		}
		updates["policy_id"] = parsed
	}

	h.db.Model(&job).Updates(updates)

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

	h.db.Preload("Schedule").Preload("Policy").Preload("Agent").Preload("StorageTarget").Preload("Connection").First(&job, job.ID)
	publishEntity(orgID.String(), realtime.TypeJobs, "updated", job.ID.String())
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
	publishEntity(orgID.String(), realtime.TypeJobs, "deleted", jobID.String())
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

	// Same in-flight guard as the scheduler: a double-clicked Run Now (or a
	// retried request after a lost response) must not fork a second run
	// while one is already active. Cancel the live run first, then retry.
	var inflight int64
	h.db.Model(&models.BackupRun{}).
		Where("backup_job_id = ? AND status IN ?", job.ID,
			[]models.BackupRunStatus{
				models.RunPending, models.RunRunning,
				models.RunUploading, models.RunVerifying,
			}).
		Count(&inflight)
	if inflight > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "a run for this job is already in progress"})
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
	publishRun(run, models.RunPending, 0, 0, 0, "", "", "")

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
	action := "disabled"
	if enabled {
		action = "enabled"
	}
	publishEntity(orgID.String(), realtime.TypeJobs, action, jobID.String())
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}
