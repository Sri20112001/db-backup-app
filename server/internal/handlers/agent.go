package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentHandler struct {
	db *gorm.DB
}

func NewAgentHandler(db *gorm.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

func (h *AgentHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var agents []models.Agent
	h.db.Where("organization_id = ?", orgID).Find(&agents)
	c.JSON(http.StatusOK, agents)
}

// GenerateRegistrationToken creates a one-time token the agent uses to register.
func (h *AgentHandler) GenerateRegistrationToken(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	token := randomHex(32)
	agent := models.Agent{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgID,
		Name:            "pending-" + randomHex(4),
		RegistrationKey: &token,
		Status:          models.AgentOffline,
	}
	if err := h.db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"agent_id":         agent.ID,
		"registration_key": token,
	})
}

// Register is called by the agent binary with the registration key.
func (h *AgentHandler) Register(c *gin.Context) {
	var req struct {
		RegistrationKey string `json:"registration_key" binding:"required"`
		Hostname        string `json:"hostname" binding:"required"`
		OS              string `json:"os"`
		Version         string `json:"version"`
		IPAddress       string `json:"ip_address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var agent models.Agent
	if err := h.db.Where("registration_key = ?", req.RegistrationKey).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid registration key"})
		return
	}

	agentToken := randomHex(48)
	updates := map[string]interface{}{
		"name":             req.Hostname,
		"token":            &agentToken,
		"version":          req.Version,
		"status":           models.AgentOnline,
		"registration_key": nil,
	}
	h.db.Model(&agent).Updates(updates)

	// Create or update machine record
	machine := models.Machine{
		OrganizationID: agent.OrganizationID,
		AgentID:        agent.ID,
		Hostname:       req.Hostname,
		OS:             req.OS,
		IPAddress:      req.IPAddress,
	}
	h.db.Where("agent_id = ?", agent.ID).Assign(machine).FirstOrCreate(&machine)

	c.JSON(http.StatusOK, gin.H{
		"agent_id":    agent.ID,
		"agent_token": agentToken,
	})
}

func (h *AgentHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var agent models.Agent
	if err := h.db.Where("id = ? AND organization_id = ?", agentID, orgID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (h *AgentHandler) Delete(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.db.Where("id = ? AND organization_id = ?", agentID, orgID).Delete(&models.Agent{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// touchAgent marks the agent online and bumps last_seen_at. Called on every
// agent-token-authenticated HTTP call so polling agents (no gRPC stream)
// stay ONLINE instead of being flagged stale by the health monitor.
func touchAgent(db *gorm.DB, agent *models.Agent) {
	now := time.Now()
	db.Model(agent).Updates(map[string]interface{}{
		"status":       models.AgentOnline,
		"last_seen_at": &now,
	})
}

// agentFromToken authenticates an agent via the Bearer agent token.
func (h *AgentHandler) agentFromToken(c *gin.Context) (*models.Agent, bool) {
	token := extractBearerToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing agent token"})
		return nil, false
	}
	var agent models.Agent
	if err := h.db.Where("token = ?", token).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return nil, false
	}
	return &agent, true
}

type agentRunDTO struct {
	ID          string `json:"id"`
	BackupJobID string `json:"backup_job_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// PendingRuns lists PENDING backup runs assigned to the calling agent.
// Polling agents use this (plus ClaimRun) instead of the gRPC stream.
func (h *AgentHandler) PendingRuns(c *gin.Context) {
	agent, ok := h.agentFromToken(c)
	if !ok {
		return
	}
	touchAgent(h.db, agent)

	var runs []models.BackupRun
	h.db.Where("agent_id = ? AND status = ?", agent.ID, models.RunPending).
		Order("created_at ASC").Find(&runs)

	out := make([]agentRunDTO, 0, len(runs))
	for _, r := range runs {
		out = append(out, agentRunDTO{
			ID:          r.ID.String(),
			BackupJobID: r.BackupJobID.String(),
			Status:      string(r.Status),
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, out)
}

type agentJobConfigDTO struct {
	JobID           string `json:"job_id"`
	Name            string `json:"name"`
	SourceType      string `json:"source_type"`
	SourcePath      string `json:"source_path"`
	SourceDatabase  string `json:"source_database"`
	IncludePatterns string `json:"include_patterns"`
	ExcludePatterns string `json:"exclude_patterns"`
	Mode            string `json:"mode"`
	Encrypted       bool   `json:"encrypted"`
	RetentionDays   int    `json:"retention_days"`
	CronExpr        string `json:"cron_expr"`
	Timezone        string `json:"timezone"`
	StorageType     string `json:"storage_type"`
	StorageBucket   string `json:"storage_bucket"`
	StorageRegion   string `json:"storage_region"`
	StorageEndpoint string `json:"storage_endpoint"`
	StoragePath     string `json:"storage_path"`
	// Note: storage credentials are intentionally omitted. The v0.1 agent
	// supports LOCAL targets only; S3/SMB will include decrypted keys here.
}

func jobConfigDTO(job *models.BackupJob) agentJobConfigDTO {
	dto := agentJobConfigDTO{
		JobID:           job.ID.String(),
		Name:            job.Name,
		SourceType:      string(job.SourceType),
		SourcePath:      job.SourcePath,
		SourceDatabase:  job.SourceDatabase,
		IncludePatterns: job.IncludePatterns,
		ExcludePatterns: job.ExcludePatterns,
		Mode:            string(job.Mode),
		Encrypted:       job.Encrypted,
		RetentionDays:   job.RetentionDays,
		StorageType:     string(job.StorageTarget.Type),
		StorageBucket:   job.StorageTarget.Bucket,
		StorageRegion:   job.StorageTarget.Region,
		StorageEndpoint: job.StorageTarget.Endpoint,
		StoragePath:     job.StorageTarget.Path,
	}
	if job.Schedule != nil {
		dto.CronExpr = job.Schedule.CronExpr
		dto.Timezone = job.Schedule.Timezone
	}
	return dto
}

// ClaimRun atomically moves a PENDING run to RUNNING for the calling agent
// and returns the run plus its job config. 409 if already claimed.
func (h *AgentHandler) ClaimRun(c *gin.Context) {
	agent, ok := h.agentFromToken(c)
	if !ok {
		return
	}
	touchAgent(h.db, agent)

	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	now := time.Now()
	res := h.db.Model(&models.BackupRun{}).
		Where("id = ? AND agent_id = ? AND status = ?", runID, agent.ID, models.RunPending).
		Updates(map[string]interface{}{"status": models.RunRunning, "started_at": &now})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claim failed"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "run already claimed or not pending"})
		return
	}

	var run models.BackupRun
	if err := h.db.Preload("BackupJob.StorageTarget").Preload("BackupJob.Schedule").
		Where("id = ?", runID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"run": agentRunDTO{
			ID:          run.ID.String(),
			BackupJobID: run.BackupJobID.String(),
			Status:      string(run.Status),
			CreatedAt:   run.CreatedAt.Format(time.RFC3339),
		},
		"config": jobConfigDTO(&run.BackupJob),
	})
}

// JobConfig returns the job + storage config for an agent-owned job.
func (h *AgentHandler) JobConfig(c *gin.Context) {
	agent, ok := h.agentFromToken(c)
	if !ok {
		return
	}
	touchAgent(h.db, agent)

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	var job models.BackupJob
	if err := h.db.Preload("StorageTarget").Preload("Schedule").
		Where("id = ? AND agent_id = ?", jobID, agent.ID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, jobConfigDTO(&job))
}

type agentRestoreDTO struct {
	ID              string `json:"id"`
	BackupRunID     string `json:"backup_run_id"`
	DestinationPath string `json:"destination_path"`
	TargetDatabase  string `json:"target_database"`
	Status          string `json:"status"`
	StoragePath     string `json:"storage_path"`
	StorageType     string `json:"storage_type"`
	CreatedAt       string `json:"created_at"`
}

// PendingRestores lists PENDING restore jobs assigned to the calling agent.
func (h *AgentHandler) PendingRestores(c *gin.Context) {
	agent, ok := h.agentFromToken(c)
	if !ok {
		return
	}
	touchAgent(h.db, agent)

	var restores []models.RestoreJob
	h.db.Preload("BackupRun").
		Where("agent_id = ? AND status = ?", agent.ID, models.RestorePending).
		Order("created_at ASC").Find(&restores)

	out := make([]agentRestoreDTO, 0, len(restores))
	for _, r := range restores {
		var target models.StorageTarget
		h.db.Where("id = ?", r.BackupRun.StorageTargetID).First(&target)
		out = append(out, restoreDTO(&r, &target))
	}
	c.JSON(http.StatusOK, out)
}

// ClaimRestore atomically moves a PENDING restore to RUNNING and returns it
// with the source backup run's storage location.
func (h *AgentHandler) ClaimRestore(c *gin.Context) {
	agent, ok := h.agentFromToken(c)
	if !ok {
		return
	}
	touchAgent(h.db, agent)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	now := time.Now()
	res := h.db.Model(&models.RestoreJob{}).
		Where("id = ? AND agent_id = ? AND status = ?", id, agent.ID, models.RestorePending).
		Updates(map[string]interface{}{"status": models.RestoreRunning, "started_at": &now})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claim failed"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "restore already claimed or not pending"})
		return
	}

	var r models.RestoreJob
	if err := h.db.Preload("BackupRun").Where("id = ?", id).First(&r).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var target models.StorageTarget
	h.db.Where("id = ?", r.BackupRun.StorageTargetID).First(&target)
	c.JSON(http.StatusOK, restoreDTO(&r, &target))
}

func restoreDTO(r *models.RestoreJob, target *models.StorageTarget) agentRestoreDTO {
	return agentRestoreDTO{
		ID:              r.ID.String(),
		BackupRunID:     r.BackupRunID.String(),
		DestinationPath: r.DestinationPath,
		TargetDatabase:  r.TargetDatabase,
		Status:          string(r.Status),
		StoragePath:     r.BackupRun.StoragePath,
		StorageType:     string(target.Type),
		CreatedAt:       r.CreatedAt.Format(time.RFC3339),
	}
}
