package handlers

import (
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AgentWorkHandler serves the HTTP polling protocol used by the backup agent:
// agents discover work via GET /api/agent/runs|restores and claim it via
// POST /api/agent/runs|restores/:id/claim. All endpoints use agent-token auth
// (X-Agent-ID header + Bearer token), not user JWT.
type AgentWorkHandler struct {
	db     *gorm.DB
	encKey []byte
}

func NewAgentWorkHandler(db *gorm.DB, encKey []byte) *AgentWorkHandler {
	return &AgentWorkHandler{db: db, encKey: encKey}
}

func (h *AgentWorkHandler) authenticateAgent(c *gin.Context) (*models.Agent, bool) {
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

// touch records a heartbeat so the health monitor (3-minute staleness
// threshold) does not flap polling agents OFFLINE. Polling agents have no
// gRPC stream, so this is their only liveness signal.
func (h *AgentWorkHandler) touch(agent *models.Agent) {
	now := time.Now()
	h.db.Model(agent).Updates(map[string]interface{}{
		"status":       models.AgentOnline,
		"last_seen_at": &now,
	})
}

// --- Backup runs ---

type agentRunDTO struct {
	ID          uuid.UUID `json:"id"`
	BackupJobID uuid.UUID `json:"backup_job_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
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
	StorageType     string `json:"storage_type"`
	StorageBucket   string `json:"storage_bucket"`
	StorageRegion   string `json:"storage_region"`
	StorageEndpoint string `json:"storage_endpoint"`
	StoragePath     string `json:"storage_path"`
}

// PendingRuns lists PENDING runs assigned to the calling agent, oldest first.
func (h *AgentWorkHandler) PendingRuns(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}
	h.touch(agent)

	var runs []models.BackupRun
	h.db.Where("agent_id = ? AND status = ?", agent.ID, models.RunPending).
		Order("created_at ASC").Find(&runs)

	out := make([]agentRunDTO, 0, len(runs))
	for _, r := range runs {
		out = append(out, agentRunDTO{
			ID:          r.ID,
			BackupJobID: r.BackupJobID,
			Status:      string(r.Status),
			CreatedAt:   r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

// ClaimRun atomically moves a PENDING run to RUNNING and returns it with the
// full job config the agent needs to execute. Returns 409 if the run is
// already claimed (or otherwise not PENDING), 404 if it is not ours.
func (h *AgentWorkHandler) ClaimRun(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}
	h.touch(agent)

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
		var run models.BackupRun
		if err := h.db.Where("id = ? AND agent_id = ?", runID, agent.ID).First(&run).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "run already claimed"})
		return
	}

	var run models.BackupRun
	if err := h.db.Preload("BackupJob.StorageTarget").
		Where("id = ?", runID).First(&run).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load run"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"run": agentRunDTO{
			ID:          run.ID,
			BackupJobID: run.BackupJobID,
			Status:      string(run.Status),
			CreatedAt:   run.CreatedAt,
		},
		"config": agentJobConfigDTO{
			JobID:           run.BackupJob.ID.String(),
			Name:            run.BackupJob.Name,
			SourceType:      string(run.BackupJob.SourceType),
			SourcePath:      run.BackupJob.SourcePath,
			SourceDatabase:  run.BackupJob.SourceDatabase,
			IncludePatterns: run.BackupJob.IncludePatterns,
			ExcludePatterns: run.BackupJob.ExcludePatterns,
			Mode:            string(run.BackupJob.Mode),
			Encrypted:       run.BackupJob.Encrypted,
			RetentionDays:   run.BackupJob.RetentionDays,
			StorageType:     string(run.BackupJob.StorageTarget.Type),
			StorageBucket:   run.BackupJob.StorageTarget.Bucket,
			StorageRegion:   run.BackupJob.StorageTarget.Region,
			StorageEndpoint: run.BackupJob.StorageTarget.Endpoint,
			StoragePath:     run.BackupJob.StorageTarget.Path,
		},
	})
}

// --- Restores ---

type agentRestoreDTO struct {
	ID              uuid.UUID `json:"id"`
	BackupRunID     uuid.UUID `json:"backup_run_id"`
	DestinationPath string    `json:"destination_path"`
	TargetDatabase  string    `json:"target_database"`
	Status          string    `json:"status"`
	StoragePath     string    `json:"storage_path"`
	StorageType     string    `json:"storage_type"`
	// DataKey is the raw base64 data key for encrypted backups, unwrapped
	// with the server ENCRYPTION_KEY. Empty for unencrypted backups.
	DataKey   string    `json:"data_key,omitempty"`
	// SourceType/SourceDatabase identify what is being restored.
	SourceType     string `json:"source_type"`
	SourceDatabase string `json:"source_database"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *AgentWorkHandler) toAgentRestoreDTO(job models.RestoreJob) agentRestoreDTO {
	dto := agentRestoreDTO{
		ID:              job.ID,
		BackupRunID:     job.BackupRunID,
		DestinationPath: job.DestinationPath,
		TargetDatabase:  job.TargetDatabase,
		Status:          string(job.Status),
		StoragePath:     job.BackupRun.StoragePath,
		StorageType:     string(job.BackupRun.BackupJob.StorageTarget.Type),
		SourceType:      string(job.BackupRun.BackupJob.SourceType),
		SourceDatabase:  job.BackupRun.BackupJob.SourceDatabase,
		CreatedAt:       job.CreatedAt,
	}
	if job.BackupRun.DataKeyEncrypted != "" {
		if raw, err := decrypt(h.encKey, job.BackupRun.DataKeyEncrypted); err == nil {
			dto.DataKey = raw
		}
	}
	return dto
}

// PendingRestores lists PENDING restores assigned to the calling agent.
func (h *AgentWorkHandler) PendingRestores(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}
	h.touch(agent)

	var jobs []models.RestoreJob
	h.db.Preload("BackupRun.BackupJob.StorageTarget").
		Where("agent_id = ? AND status = ?", agent.ID, models.RestorePending).
		Order("created_at ASC").Find(&jobs)

	out := make([]agentRestoreDTO, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, h.toAgentRestoreDTO(j))
	}
	c.JSON(http.StatusOK, out)
}

// ClaimRestore atomically moves a PENDING restore to RUNNING. Returns 409 if
// already claimed, 404 if it is not ours.
func (h *AgentWorkHandler) ClaimRestore(c *gin.Context) {
	agent, ok := h.authenticateAgent(c)
	if !ok {
		return
	}
	h.touch(agent)

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
		var job models.RestoreJob
		if err := h.db.Where("id = ? AND agent_id = ?", id, agent.ID).First(&job).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "restore already claimed"})
		return
	}

	var job models.RestoreJob
	if err := h.db.Preload("BackupRun.BackupJob.StorageTarget").
		Where("id = ?", id).First(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load restore"})
		return
	}
	c.JSON(http.StatusOK, h.toAgentRestoreDTO(job))
}
