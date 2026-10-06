package handlers

import (
	"net/http"
	"strings"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackupPolicyHandler manages reusable how-to-back-up configurations.
// Policies never touch secrets: they hold schedule/processing/retention/
// verification/retry targets only. Jobs attach via policy_id (NULL = inline).
type BackupPolicyHandler struct {
	db *gorm.DB
}

func NewBackupPolicyHandler(db *gorm.DB) *BackupPolicyHandler {
	return &BackupPolicyHandler{db: db}
}

type policyRequest struct {
	Name                string `json:"name"`
	Strategy            string `json:"strategy"`
	CronExpr            string `json:"cron_expr"`
	Timezone            string `json:"timezone"`
	Enabled             *bool  `json:"enabled"`
	Mode                string `json:"mode"`
	Encrypted           *bool  `json:"encrypted"`
	RetentionDays       *int   `json:"retention_days"`
	VerificationEnabled *bool  `json:"verification_enabled"`
	MaxRetries          *int   `json:"max_retries"`
	RetryDelaySeconds   *int   `json:"retry_delay_seconds"`
	SLATargetMinutes    *int   `json:"sla_target_minutes"`
	RPOTargetMinutes    *int   `json:"rpo_target_minutes"`
	RTOTargetMinutes    *int   `json:"rto_target_minutes"`
}

func boolOr(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

func intOr(i *int, def int) int {
	if i == nil {
		return def
	}
	return *i
}

// validatePolicy enforces only executable configurations: FULL-only strategy
// (never fake incrementals), non-negative bounds, cron parse.
func validatePolicy(req *policyRequest) (models.BackupPolicy, error) {
	p := models.BackupPolicy{}
	if strings.TrimSpace(req.Name) == "" {
		return p, errBadPolicy("name is required")
	}
	strategy := strings.ToUpper(strings.TrimSpace(req.Strategy))
	if strategy == "" {
		strategy = models.StrategyFull
	}
	if strategy != models.StrategyFull {
		return p, errBadPolicy("strategy must be FULL (incremental/differential are not implemented)")
	}
	mode := models.BackupMode(strings.ToUpper(strings.TrimSpace(req.Mode)))
	if mode == "" {
		mode = models.ModeNormal
	}
	if mode != models.ModeNormal && mode != models.ModeCompressed {
		return p, errBadPolicy("mode must be NORMAL or COMPRESSED")
	}
	retention := intOr(req.RetentionDays, 30)
	if retention < 0 || retention > 3650 {
		return p, errBadPolicy("retention_days must be 0-3650")
	}
	maxRetries := intOr(req.MaxRetries, 0)
	if maxRetries < 0 || maxRetries > 10 {
		return p, errBadPolicy("max_retries must be 0-10")
	}
	delay := intOr(req.RetryDelaySeconds, 300)
	if delay < 0 || delay > 86400 {
		return p, errBadPolicy("retry_delay_seconds must be 0-86400")
	}
	for _, v := range []*int{req.SLATargetMinutes, req.RPOTargetMinutes, req.RTOTargetMinutes} {
		if v != nil && (*v < 0 || *v > 525600) {
			return p, errBadPolicy("SLA/RPO/RTO targets must be 0-525600 minutes")
		}
	}
	cron := strings.TrimSpace(req.CronExpr)
	if cron != "" {
		if _, err := cronParser.Parse(cron); err != nil {
			return p, errBadPolicy("invalid cron_expr: " + err.Error())
		}
	}
	tz := strings.TrimSpace(req.Timezone)
	if tz == "" {
		tz = "UTC"
	}
	p.Name = strings.TrimSpace(req.Name)
	p.Strategy = strategy
	p.CronExpr = cron
	p.Timezone = tz
	p.Enabled = boolOr(req.Enabled, true)
	p.Mode = mode
	p.Encrypted = boolOr(req.Encrypted, false)
	p.RetentionDays = retention
	p.VerificationEnabled = boolOr(req.VerificationEnabled, true)
	p.MaxRetries = maxRetries
	p.RetryDelaySeconds = delay
	p.SLATargetMinutes = intOr(req.SLATargetMinutes, 0)
	p.RPOTargetMinutes = intOr(req.RPOTargetMinutes, 0)
	p.RTOTargetMinutes = intOr(req.RTOTargetMinutes, 0)
	return p, nil
}

type policyError string

func (e policyError) Error() string { return string(e) }

func errBadPolicy(msg string) error { return policyError(msg) }

func (h *BackupPolicyHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var policies []models.BackupPolicy
	h.db.Where("organization_id = ?", orgID).Order("created_at ASC").Find(&policies)
	c.JSON(http.StatusOK, asArray(policies))
}

func (h *BackupPolicyHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var p models.BackupPolicy
	if err := h.db.Where("id = ? AND organization_id = ?", id, orgID).First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var jobCount int64
	h.db.Model(&models.BackupJob{}).Where("policy_id = ? AND organization_id = ?", id, orgID).Count(&jobCount)
	c.JSON(http.StatusOK, gin.H{"policy": p, "attached_jobs": jobCount})
}

func (h *BackupPolicyHandler) Create(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := validatePolicy(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.Base = models.Base{ID: uuid.New()}
	p.OrganizationID = orgID
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishEntity(orgID.String(), realtime.TypePolicies, "created", p.ID.String())
	c.JSON(http.StatusCreated, p)
}

func (h *BackupPolicyHandler) Update(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var p models.BackupPolicy
	if err := h.db.Where("id = ? AND organization_id = ?", id, orgID).First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := validatePolicy(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated.ID = p.ID
	updated.OrganizationID = orgID
	updated.CreatedAt = p.CreatedAt
	if err := h.db.Model(&p).Updates(map[string]interface{}{
		"name": updated.Name, "strategy": updated.Strategy,
		"cron_expr": updated.CronExpr, "timezone": updated.Timezone, "enabled": updated.Enabled,
		"mode": updated.Mode, "encrypted": updated.Encrypted, "retention_days": updated.RetentionDays,
		"verification_enabled": updated.VerificationEnabled,
		"max_retries": updated.MaxRetries, "retry_delay_seconds": updated.RetryDelaySeconds,
		"sla_target_minutes": updated.SLATargetMinutes,
		"rpo_target_minutes": updated.RPOTargetMinutes,
		"rto_target_minutes": updated.RTOTargetMinutes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	h.db.Where("id = ?", p.ID).First(&p)
	publishEntity(orgID.String(), realtime.TypePolicies, "updated", p.ID.String())
	c.JSON(http.StatusOK, p)
}

func (h *BackupPolicyHandler) Delete(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Detach instead of blocking: attached jobs fall back to their inline
	// settings (which were always preserved), so deletion never orphans work.
	h.db.Model(&models.BackupJob{}).
		Where("policy_id = ? AND organization_id = ?", id, orgID).
		Update("policy_id", nil)
	// RowsAffected distinguishes "deleted" from "never existed here": other
	// orgs' policies (and ghosts) answer 404, never a misleading 204.
	res := h.db.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.BackupPolicy{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	publishEntity(orgID.String(), realtime.TypePolicies, "deleted", id.String())
	c.JSON(http.StatusNoContent, nil)
}
