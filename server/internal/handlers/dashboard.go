package handlers

import (
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
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

	// SLA summary: count jobs that missed their SLA window in the last 24h.
	slaBreaches := h.countSLABreaches(orgID, since)

	// Size anomalies: runs whose uploaded size deviates >50% from the job baseline.
	anomalies := h.detectSizeAnomalies(orgID)

	c.JSON(http.StatusOK, gin.H{
		"total_jobs":      totalJobs,
		"enabled_jobs":    enabledJobs,
		"agents_online":   agentsOnline,
		"agents_offline":  agentsOffline,
		"successful_runs": successfulRuns,
		"failed_runs":     failedRuns,
		"total_bytes":     totalBytes,
		"recent_runs":     asArray(recentRuns),
		"sla_breaches_24h": slaBreaches,
		"size_anomalies":  anomalies,
	})
}

// countSLABreaches returns the number of completed runs in the window that
// exceeded their effective SLATargetMinutes (start-to-completion duration).
func (h *DashboardHandler) countSLABreaches(orgID uuid.UUID, since time.Time) int {
	var jobs []models.BackupJob
	h.db.Preload("Policy").Where("organization_id = ?", orgID).Find(&jobs)
	breaches := 0
	for _, job := range jobs {
		if target := models.EffectivePolicy(&job).SLATargetMinutes; target > 0 {
			var runs []models.BackupRun
			h.db.Where("backup_job_id = ? AND status = ? AND completed_at > ?", job.ID, models.RunCompleted, since).Find(&runs)
			for _, r := range runs {
				if r.DurationSeconds > int64(target)*60 {
					breaches++
				}
			}
		}
	}
	return breaches
}

// sizeAnomalyDTO describes a single backup run whose size is anomalous.
type sizeAnomalyDTO struct {
	JobID      uuid.UUID `json:"job_id"`
	JobName    string    `json:"job_name"`
	RunID      uuid.UUID `json:"run_id"`
	ActualBytes int64    `json:"actual_bytes"`
	AvgBytes   int64     `json:"avg_bytes"`
	PctChange  int       `json:"pct_change"` // negative = smaller than baseline
}

// detectSizeAnomalies returns runs from the last 7 days whose size deviates
// more than 50% from the rolling average stored in BackupSizeBaseline.
func (h *DashboardHandler) detectSizeAnomalies(orgID uuid.UUID) []sizeAnomalyDTO {
	var baselines []models.BackupSizeBaseline
	h.db.Joins("JOIN backup_jobs ON backup_jobs.id = backup_size_baselines.backup_job_id").
		Where("backup_jobs.organization_id = ? AND backup_size_baselines.avg_bytes > 0 AND backup_size_baselines.sample_count >= 3", orgID).
		Find(&baselines)

	result := []sizeAnomalyDTO{}
	since := time.Now().Add(-7 * 24 * time.Hour)
	for _, b := range baselines {
		var runs []models.BackupRun
		h.db.Preload("BackupJob").
			Where("backup_job_id = ? AND status = ? AND completed_at > ?", b.BackupJobID, models.RunCompleted, since).
			Find(&runs)
		for _, r := range runs {
			if b.AvgBytes == 0 {
				continue
			}
			pct := int((r.BytesUploaded - b.AvgBytes) * 100 / b.AvgBytes)
			if pct < -50 || pct > 200 {
				result = append(result, sizeAnomalyDTO{
					JobID:       b.BackupJobID,
					JobName:     r.BackupJob.Name,
					RunID:       r.ID,
					ActualBytes: r.BytesUploaded,
					AvgBytes:    b.AvgBytes,
					PctChange:   pct,
				})
			}
		}
	}
	return result
}

var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func (h *DashboardHandler) BackupHealth(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	var jobs []models.BackupJob
	h.db.Preload("Schedule").Preload("Policy").Where("organization_id = ? AND enabled = true", orgID).Find(&jobs)

	type jobHealth struct {
		JobID            uuid.UUID  `json:"job_id"`
		JobName          string     `json:"job_name"`
		Status           string     `json:"status"`
		LastSuccessAt    *time.Time `json:"last_success_at"`
		LastFailureAt    *time.Time `json:"last_failure_at"`
		RecoveryPoints   int64      `json:"recovery_points"`
		NextRunAt        *time.Time `json:"next_run_at"`
		// SLA fields
		SLATargetMinutes int        `json:"sla_target_minutes"`
		LastDurationSecs int64      `json:"last_duration_seconds"`
		SLAStatus        string     `json:"sla_status"` // OK | BREACH | UNKNOWN
		// RPO fields
		RPOTargetMinutes int        `json:"rpo_target_minutes"`
		ActualRPOMinutes int        `json:"actual_rpo_minutes"` // minutes since last success
		RPOStatus        string     `json:"rpo_status"`         // OK | BREACH | UNKNOWN
		// RTO fields
		RTOTargetMinutes  int       `json:"rto_target_minutes"`
		LastRestoreSecs   int64     `json:"last_restore_seconds"`
		RTOStatus         string    `json:"rto_status"` // OK | BREACH | UNKNOWN | NO_DATA
		// Size anomaly
		SizeAnomalyPct *int        `json:"size_anomaly_pct,omitempty"`
	}

	result := make([]jobHealth, 0, len(jobs))
	for _, job := range jobs {
		// Effective settings: a disabled policy pauses health tracking just
		// like a disabled job; schedule and SLA/RPO/RTO targets resolve
		// policy-first so the dashboard never reports stale inline values.
		eff := models.EffectivePolicy(&job)
		if !eff.Enabled {
			continue
		}
		var lastSuccess, lastFailure models.BackupRun
		h.db.Where("backup_job_id = ? AND status = ?", job.ID, models.RunCompleted).
			Order("completed_at DESC").First(&lastSuccess)
		h.db.Where("backup_job_id = ? AND status = ?", job.ID, models.RunFailed).
			Order("created_at DESC").First(&lastFailure)

		var count int64
		h.db.Model(&models.BackupRun{}).
			Where("backup_job_id = ? AND status = ?", job.ID, models.RunCompleted).Count(&count)

		// Overall health status
		status := "HEALTHY"
		if lastSuccess.ID == uuid.Nil {
			status = "UNKNOWN"
		} else if lastFailure.ID != uuid.Nil && lastFailure.CreatedAt.After(lastSuccess.CreatedAt) {
			status = "FAILED"
		} else if lastSuccess.CompletedAt != nil && time.Since(*lastSuccess.CompletedAt) > 25*time.Hour {
			status = "WARNING"
		}

		// Next scheduled run
		var nextRun *time.Time
		if eff.CronExpr != "" {
			if sched, err := cronParser.Parse(eff.CronExpr); err == nil {
				loc := time.UTC
				if eff.Timezone != "" {
					if l, err := time.LoadLocation(eff.Timezone); err == nil {
						loc = l
					}
				}
				t := sched.Next(time.Now().In(loc))
				nextRun = &t
			}
		}

		// SLA tracking
		slaStatus := "UNKNOWN"
		if eff.SLATargetMinutes > 0 && lastSuccess.ID != uuid.Nil {
			if lastSuccess.DurationSeconds > int64(eff.SLATargetMinutes)*60 {
				slaStatus = "BREACH"
			} else {
				slaStatus = "OK"
			}
		}

		// RPO tracking: minutes since last successful backup
		actualRPO := 0
		rpoStatus := "UNKNOWN"
		if lastSuccess.CompletedAt != nil {
			actualRPO = int(time.Since(*lastSuccess.CompletedAt).Minutes())
			if eff.RPOTargetMinutes > 0 {
				if actualRPO > eff.RPOTargetMinutes {
					rpoStatus = "BREACH"
				} else {
					rpoStatus = "OK"
				}
			}
		}

		// RTO tracking: last restore duration for this job's runs
		rtoStatus := "NO_DATA"
		var lastRestoreSecs int64
		var lastRestore models.RestoreJob
		h.db.Joins("JOIN backup_runs ON backup_runs.id = restore_jobs.backup_run_id").
			Where("backup_runs.backup_job_id = ? AND restore_jobs.status = ?", job.ID, models.RestoreCompleted).
			Order("restore_jobs.completed_at DESC").
			First(&lastRestore)
		if lastRestore.ID != uuid.Nil {
			lastRestoreSecs = lastRestore.DurationSeconds
			if eff.RTOTargetMinutes > 0 {
				if lastRestoreSecs > int64(eff.RTOTargetMinutes)*60 {
					rtoStatus = "BREACH"
				} else {
					rtoStatus = "OK"
				}
			} else {
				rtoStatus = "UNKNOWN"
			}
		}

		// Size anomaly for this job
		var sizeAnomalyPct *int
		var baseline models.BackupSizeBaseline
		if h.db.Where("backup_job_id = ? AND sample_count >= 3 AND avg_bytes > 0", job.ID).First(&baseline).Error == nil {
			if lastSuccess.ID != uuid.Nil && baseline.AvgBytes > 0 {
				pct := int((lastSuccess.BytesUploaded - baseline.AvgBytes) * 100 / baseline.AvgBytes)
				if pct < -50 || pct > 200 {
					sizeAnomalyPct = &pct
				}
			}
		}

		jh := jobHealth{
			JobID:            job.ID,
			JobName:          job.Name,
			Status:           status,
			RecoveryPoints:   count,
			NextRunAt:        nextRun,
			SLATargetMinutes: eff.SLATargetMinutes,
			LastDurationSecs: lastSuccess.DurationSeconds,
			SLAStatus:        slaStatus,
			RPOTargetMinutes: eff.RPOTargetMinutes,
			ActualRPOMinutes: actualRPO,
			RPOStatus:        rpoStatus,
			RTOTargetMinutes: eff.RTOTargetMinutes,
			LastRestoreSecs:  lastRestoreSecs,
			RTOStatus:        rtoStatus,
			SizeAnomalyPct:   sizeAnomalyPct,
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
