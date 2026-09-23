package handlers

import (
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) Overview(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	var totalJobs, enabledJobs int64
	h.db.Model(&models.BackupJob{}).Where("organization_id = ?", orgID).Count(&totalJobs)
	h.db.Model(&models.BackupJob{}).Where("organization_id = ? AND enabled = true", orgID).Count(&enabledJobs)

	var agentsOnline, agentsOffline int64
	h.db.Model(&models.Agent{}).Where("organization_id = ? AND status = ?", orgID, models.AgentOnline).Count(&agentsOnline)
	h.db.Model(&models.Agent{}).Where("organization_id = ? AND status = ?", orgID, models.AgentOffline).Count(&agentsOffline)

	var successfulRuns, failedRuns int64
	since := time.Now().Add(-24 * time.Hour)
	h.db.Model(&models.BackupRun{}).Where("organization_id = ? AND status = ? AND created_at > ?", orgID, models.RunCompleted, since).Count(&successfulRuns)
	h.db.Model(&models.BackupRun{}).Where("organization_id = ? AND status = ? AND created_at > ?", orgID, models.RunFailed, since).Count(&failedRuns)

	var totalBytes int64
	h.db.Model(&models.BackupRun{}).
		Where("organization_id = ? AND status = ?", orgID, models.RunCompleted).
		Select("COALESCE(SUM(bytes_uploaded), 0)").Scan(&totalBytes)

	var recentRuns []models.BackupRun
	h.db.Preload("BackupJob").
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(10).
		Find(&recentRuns)

	c.JSON(http.StatusOK, gin.H{
		"total_jobs":       totalJobs,
		"enabled_jobs":     enabledJobs,
		"agents_online":    agentsOnline,
		"agents_offline":   agentsOffline,
		"successful_runs":  successfulRuns,
		"failed_runs":      failedRuns,
		"total_bytes":      totalBytes,
		"recent_runs":      asArray(recentRuns),
	})
}

func (h *DashboardHandler) BackupHealth(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	var jobs []models.BackupJob
	h.db.Preload("Schedule").Where("organization_id = ? AND enabled = true", orgID).Find(&jobs)

	type jobHealth struct {
		JobID           uuid.UUID  `json:"job_id"`
		JobName         string     `json:"job_name"`
		Status          string     `json:"status"`
		LastSuccessAt   *time.Time `json:"last_success_at"`
		LastFailureAt   *time.Time `json:"last_failure_at"`
		RecoveryPoints  int64      `json:"recovery_points"`
	}

	result := make([]jobHealth, 0, len(jobs))
	for _, job := range jobs {
		var lastSuccess, lastFailure models.BackupRun
		h.db.Where("backup_job_id = ? AND status = ?", job.ID, models.RunCompleted).
			Order("completed_at DESC").First(&lastSuccess)
		h.db.Where("backup_job_id = ? AND status = ?", job.ID, models.RunFailed).
			Order("created_at DESC").First(&lastFailure)

		var count int64
		h.db.Model(&models.BackupRun{}).
			Where("backup_job_id = ? AND status = ?", job.ID, models.RunCompleted).Count(&count)

		status := "HEALTHY"
		if lastSuccess.ID == uuid.Nil {
			status = "UNKNOWN"
		} else if lastFailure.ID != uuid.Nil && lastFailure.CreatedAt.After(lastSuccess.CreatedAt) {
			status = "FAILED"
		} else if lastSuccess.CompletedAt != nil && time.Since(*lastSuccess.CompletedAt) > 25*time.Hour {
			status = "WARNING"
		}

		jh := jobHealth{
			JobID:          job.ID,
			JobName:        job.Name,
			Status:         status,
			RecoveryPoints: count,
		}
		if lastSuccess.CompletedAt != nil {
			jh.LastSuccessAt = lastSuccess.CompletedAt
		}
		if lastFailure.ID != uuid.Nil {
			jh.LastFailureAt = &lastFailure.CreatedAt
		}
		result = append(result, jh)
	}

	c.JSON(http.StatusOK, result)
}
