package services

import (
	"os"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type HealthMonitor struct {
	db       *gorm.DB
	mailer   *Mailer
	interval time.Duration
	stop     chan struct{}
}

func NewHealthMonitor(db *gorm.DB, mailer *Mailer) *HealthMonitor {
	return &HealthMonitor{
		db:       db,
		mailer:   mailer,
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
		// Cooldown: flapping agents must not spam an alert (and email) on
		// every 5-minute monitor tick. One AGENT_OFFLINE per agent per hour.
		var recent int64
		h.db.Model(&models.Alert{}).
			Where("type = ? AND agent_id = ? AND created_at > ?", models.AlertAgentOffline, agentID, time.Now().Add(-time.Hour)).
			Count(&recent)
		if recent > 0 {
			continue
		}
		alert := models.Alert{
			OrganizationID: agent.OrganizationID,
			Type:           models.AlertAgentOffline,
			Title:          "Agent Offline",
			Message:        "Agent " + agent.Name + " stopped reporting",
			AgentID:        &agentID,
		}
		h.db.Create(&alert)
		realtime.Publish(realtime.DefaultHub, agent.OrganizationID.String(), realtime.Event{
			Type:    realtime.TypePresence,
			Payload: map[string]interface{}{"id": agent.ID.String(), "name": agent.Name, "status": string(models.AgentOffline)},
		})
		realtime.Publish(realtime.DefaultHub, agent.OrganizationID.String(), realtime.Event{
			Type:    realtime.TypeAlert,
			Payload: alert,
		})
		NotifyOrg(h.db, h.mailer, agent.OrganizationID,
			"[VaultGuard] Agent Offline: "+agent.Name,
			"Agent "+agent.Name+" stopped reporting.\n\nScheduled backups for its jobs will be missed until it reconnects.")
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
				alert := models.Alert{
					OrganizationID: job.OrganizationID,
					Type:           models.AlertBackupMissed,
					Title:          "Backup Missed",
					Message:        "No successful backup within expected window for job: " + job.Name,
					BackupJobID:    &jobID,
				}
				h.db.Create(&alert)
				realtime.Publish(realtime.DefaultHub, job.OrganizationID.String(), realtime.Event{
					Type:    realtime.TypeAlert,
					Payload: alert,
				})
				NotifyOrg(h.db, h.mailer, job.OrganizationID,
					"[VaultGuard] Backup Missed: "+job.Name,
					"No successful backup within the expected window for job: "+job.Name+".\n\nCheck the agent and storage target.")
				log.Warn().Str("job_id", job.ID.String()).Msg("missed backup detected")
			}
		}
	}
}

// enforceRetention deletes backup runs (and their artifacts/chunks) older
// than retention_days — including the actual archive files for LOCAL storage
// targets. S3/SMB targets have no server-side client yet, so their files are
// left in place and flagged loudly: configure bucket lifecycle/Object Lock
// expiry there until remote deletion lands.
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
			h.deleteRunFiles(run)
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

// deleteRunFiles removes the on-disk archives for an expired run when the
// storage target is LOCAL (the only type the server can reach). Missing
// files are fine (already cleaned); failures are logged, never fatal — the
// DB rows still go away so history stays truthful about retention.
func (h *HealthMonitor) deleteRunFiles(run models.BackupRun) {
	var target models.StorageTarget
	if err := h.db.Where("id = ?", run.StorageTargetID).First(&target).Error; err != nil {
		log.Warn().Err(err).Str("run_id", run.ID.String()).
			Msg("retention: storage target gone, skipping file delete")
		return
	}
	if target.Type != models.StorageLocal {
		log.Warn().
			Str("run_id", run.ID.String()).
			Str("storage", string(target.Type)).
			Str("path", run.StoragePath).
			Msg("retention: remote file left in place (no server-side client) — expire it with bucket lifecycle rules")
		return
	}
	seen := map[string]bool{}
	paths := []string{run.StoragePath}
	var artifacts []models.BackupArtifact
	h.db.Where("backup_run_id = ?", run.ID).Find(&artifacts)
	for _, a := range artifacts {
		paths = append(paths, a.StoragePath)
	}
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", p).Msg("retention: file delete failed")
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
