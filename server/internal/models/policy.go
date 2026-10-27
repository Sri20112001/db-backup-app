package models

import (
	"github.com/google/uuid"
)

// --- BackupPolicy ---
//
// BackupPolicy defines HOW a backup operates (schedule, processing,
// retention, verification, retry); BackupJob defines WHAT/WHERE (source,
// agent, storage). Jobs with PolicyID == NULL use their own inline settings,
// so every existing job keeps working with no data migration.
//
// GFS-ready: retention stays day-based for now; per-tier keeps
// (hourly/daily/weekly/monthly/yearly counts) can extend this struct later
// without touching jobs or runs.

type BackupPolicy struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string    `gorm:"not null" json:"name"`
	// Strategy is FULL-only until the engines implement real incrementals.
	Strategy       string    `gorm:"not null;default:'FULL'" json:"strategy"`
	// Schedule. Empty CronExpr = manual-only policy.
	CronExpr       string    `gorm:"default:''" json:"cron_expr"`
	Timezone       string    `gorm:"default:'UTC'" json:"timezone"`
	Enabled        bool      `gorm:"default:true" json:"enabled"`
	// Processing (mirrors the job-inline equivalents they override).
	Mode           BackupMode `gorm:"not null;default:'NORMAL'" json:"mode"`
	Encrypted      bool       `gorm:"default:false" json:"encrypted"`
	RetentionDays  int        `gorm:"default:30" json:"retention_days"`
	// Verification: when true, LOCAL artifacts are server-verified after
	// COMPLETED and only verified points are restore-eligible. Remote
	// targets rely on the agent-side HEAD/checksum gate that already
	// precedes every COMPLETED report.
	VerificationEnabled bool `gorm:"default:true" json:"verification_enabled"`
	// Retry: failed runs are refired as NEW linked runs (terminal states
	// never transition out), capped at MaxRetries with DelaySeconds between
	// attempts. 0 disables. Observable via BackupRun.RetryOfRunID/RetryAttempt.
	MaxRetries        int `gorm:"default:0" json:"max_retries"`
	RetryDelaySeconds int `gorm:"default:300" json:"retry_delay_seconds"`
	// SLA / RPO / RTO targets (0 = not configured). Same semantics as the
	// job-inline equivalents they override.
	SLATargetMinutes int `gorm:"default:0" json:"sla_target_minutes"`
	RPOTargetMinutes int `gorm:"default:0" json:"rpo_target_minutes"`
	RTOTargetMinutes int `gorm:"default:0" json:"rto_target_minutes"`
}

// ResolvedPolicy is the effective how-to-back-up for one job: policy values
// when attached, otherwise the job's own inline settings. Every execution
// path (scheduler, RunNow, claim config, preflight) resolves through here
// so attached and legacy jobs behave identically downstream.
type ResolvedPolicy struct {
	CronExpr            string
	Timezone            string
	Enabled             bool
	Mode                BackupMode
	Encrypted           bool
	RetentionDays       int
	VerificationEnabled bool
	MaxRetries          int
	RetryDelaySeconds   int
	SLATargetMinutes    int
	RPOTargetMinutes    int
	RTOTargetMinutes    int
	FromPolicy          bool
	PolicyID            *uuid.UUID
}

// EffectivePolicy resolves a job's operating settings. Schedules: a policy
// with a non-empty CronExpr wins; otherwise the job's BackupSchedule row.
// Processing/retention/SLA: policy wins wholesale when attached.
func EffectivePolicy(job *BackupJob) ResolvedPolicy {
	if job.Policy != nil && job.PolicyID != nil {
		p := job.Policy
		tz := p.Timezone
		if tz == "" {
			tz = "UTC"
		}
		r := ResolvedPolicy{
			Mode:                p.Mode,
			Encrypted:           p.Encrypted,
			RetentionDays:       p.RetentionDays,
			VerificationEnabled: p.VerificationEnabled,
			MaxRetries:          p.MaxRetries,
			RetryDelaySeconds:   p.RetryDelaySeconds,
			SLATargetMinutes:    p.SLATargetMinutes,
			RPOTargetMinutes:    p.RPOTargetMinutes,
			RTOTargetMinutes:    p.RTOTargetMinutes,
			Enabled:             p.Enabled,
			FromPolicy:          true,
			PolicyID:            job.PolicyID,
		}
		if p.CronExpr != "" {
			r.CronExpr, r.Timezone = p.CronExpr, tz
		} else if job.Schedule != nil {
			// Attached policy without its own schedule: keep the job's
			// schedule row, but the policy's Enabled still governs.
			r.CronExpr, r.Timezone = job.Schedule.CronExpr, job.Schedule.Timezone
		} else {
			r.CronExpr = ""
		}
		if r.Timezone == "" {
			r.Timezone = "UTC"
		}
		return r
	}
	tz, cron := "UTC", ""
	enabled := job.Enabled
	if job.Schedule != nil {
		cron, tz = job.Schedule.CronExpr, job.Schedule.Timezone
		if tz == "" {
			tz = "UTC"
		}
	}
	return ResolvedPolicy{
		CronExpr:            cron,
		Timezone:            tz,
		Enabled:             enabled,
		Mode:                job.Mode,
		Encrypted:           job.Encrypted,
		RetentionDays:       job.RetentionDays,
		VerificationEnabled: true,
		MaxRetries:          0,
		RetryDelaySeconds:   300,
		SLATargetMinutes:    job.SLATargetMinutes,
		RPOTargetMinutes:    job.RPOTargetMinutes,
		RTOTargetMinutes:    job.RTOTargetMinutes,
		FromPolicy:          false,
	}
}
