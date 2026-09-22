package services

import (
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type HealthMonitor struct {
	db       *gorm.DB
	interval time.Duration
	stop     chan struct{}
}

func NewHealthMonitor(db *gorm.DB) *HealthMonitor {
	return &HealthMonitor{
		db:       db,
		interval: 5 * time.Minute,
		stop:     make(chan struct{}),
	}
}

func (h *HealthMonitor) Start() { go h.run() }
func (h *HealthMonitor) Stop()  { close(h.stop) }

func (h *HealthMonitor) run() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.checkAgents()
			h.checkMissedBackups()
			h.enforceRetention()
		case <-h.stop:
			return
		}
	}
}

func (h *HealthMonitor) checkAgents() {
	threshold := time.Now().Add(-3 * time.Minute)
	var staleAgents []models.Agent
	h.db.Where("status = ? AND last_seen_at < ?", models.AgentOnline, threshold).Find(&staleAgents)

	for _, agent := range staleAgents {
		h.db.Model(&agent).Update("status", models.AgentOffline)
		agentID := agent.ID
		h.db.Create(&models.Alert{
			OrganizationID: agent.OrganizationID,
			Type:           models.AlertAgentOffline,
			Title:          "Agent Offline",
			Message:        "Agent " + agent.Name + " stopped reporting",
			AgentID:        &agentID,
		})
		log.Warn().Str("agent_id", agent.ID.String()).Msg("agent marked offline")
	}
}

func (h *HealthMonitor) checkMissedBackups() {
	var jobs []models.BackupJob
	h.db.Preload("Schedule").Where("enabled = true").Find(&jobs)

	for _, job := range jobs {
		if job.Schedule == nil {
			continue
		}

		// Use 2x the expected interval as the missed-backup window.
		// Default to 25h if we can't parse the cron expression.
		window := missedWindow(job.Schedule.CronExpr)
		since := time.Now().Add(-window)

		var count int64
		h.db.Model(&models.BackupRun{}).
			Where("backup_job_id = ? AND status = ? AND completed_at > ?", job.ID, models.RunCompleted, since).
			Count(&count)

		if count == 0 {
			var existing int64
			h.db.Model(&models.Alert{}).
				Where("type = ? AND backup_job_id = ? AND created_at > ?", models.AlertBackupMissed, job.ID, since).
				Count(&existing)
			if existing == 0 {
				jobID := job.ID
				h.db.Create(&models.Alert{
					OrganizationID: job.OrganizationID,
					Type:           models.AlertBackupMissed,
					Title:          "Backup Missed",
					Message:        "No successful backup within expected window for job: " + job.Name,
					BackupJobID:    &jobID,
				})
				log.Warn().Str("job_id", job.ID.String()).Msg("missed backup detected")
			}
		}
	}
}

// enforceRetention deletes backup runs (and their artifacts/chunks) older than retention_days.
func (h *HealthMonitor) enforceRetention() {
	var jobs []models.BackupJob
	h.db.Where("retention_days > 0").Find(&jobs)

	for _, job := range jobs {
		cutoff := time.Now().AddDate(0, 0, -job.RetentionDays)

		// Find expired completed runs
		var expiredRuns []models.BackupRun
		h.db.Where("backup_job_id = ? AND status = ? AND completed_at < ?",
			job.ID, models.RunCompleted, cutoff).Find(&expiredRuns)

		for _, run := range expiredRuns {
			// Delete chunks → artifacts → run in order
			var artifacts []models.BackupArtifact
			h.db.Where("backup_run_id = ?", run.ID).Find(&artifacts)
			for _, a := range artifacts {
				h.db.Where("artifact_id = ?", a.ID).Delete(&models.BackupChunk{})
			}
			h.db.Where("backup_run_id = ?", run.ID).Delete(&models.BackupArtifact{})
			h.db.Delete(&run)
			log.Info().Str("run_id", run.ID.String()).
				Str("job_id", job.ID.String()).
				Msg("retention: deleted expired backup run")
		}
	}
}

// missedWindow returns a sensible alert window based on the cron expression.
// It uses simple prefix matching for common patterns; defaults to 25h.
func missedWindow(cron string) time.Duration {
	// Common cron patterns: "@hourly", "@daily", "*/N * * * *" (every N minutes)
	switch cron {
	case "@hourly":
		return 2 * time.Hour
	case "@daily", "0 0 * * *", "@midnight":
		return 25 * time.Hour
	case "@weekly":
		return 8 * 24 * time.Hour
	case "@monthly":
		return 32 * 24 * time.Hour
	}
	// Default: 25 hours covers most daily jobs
	return 25 * time.Hour
}
