package handlers

import (
	"strings"
	"testing"

	"github.com/backup-saas/server/internal/models"
	"github.com/google/uuid"
)

// --- validatePolicy: only executable configurations ---

func TestValidatePolicyDefaults(t *testing.T) {
	p, err := validatePolicy(&policyRequest{Name: "  Nightly  "})
	if err != nil {
		t.Fatalf("valid minimal policy rejected: %v", err)
	}
	if p.Name != "Nightly" {
		t.Errorf("name not trimmed: %q", p.Name)
	}
	if p.Strategy != models.StrategyFull || p.Mode != models.ModeNormal {
		t.Errorf("bad defaults: %+v", p)
	}
	if p.Timezone != "UTC" || p.RetentionDays != 30 || !p.Enabled || !p.VerificationEnabled {
		t.Errorf("bad defaults: %+v", p)
	}
	if p.MaxRetries != 0 || p.RetryDelaySeconds != 300 {
		t.Errorf("retry must default off: %+v", p)
	}
}

func TestValidatePolicyRejects(t *testing.T) {
	cases := []struct {
		name string
		req  policyRequest
		want string
	}{
		{"empty name", policyRequest{}, "name is required"},
		{"blank name", policyRequest{Name: "  "}, "name is required"},
		{"incremental strategy", policyRequest{Name: "x", Strategy: "incremental"}, "FULL"},
		{"differential strategy", policyRequest{Name: "x", Strategy: "DIFFERENTIAL"}, "FULL"},
		{"bad mode", policyRequest{Name: "x", Mode: "ULTRA"}, "NORMAL or COMPRESSED"},
		{"negative retention", policyRequest{Name: "x", RetentionDays: intPtr(-1)}, "retention_days"},
		{"huge retention", policyRequest{Name: "x", RetentionDays: intPtr(99999)}, "retention_days"},
		{"negative retries", policyRequest{Name: "x", MaxRetries: intPtr(-1)}, "max_retries"},
		{"excessive retries", policyRequest{Name: "x", MaxRetries: intPtr(11)}, "max_retries"},
		{"negative delay", policyRequest{Name: "x", RetryDelaySeconds: intPtr(-5)}, "retry_delay_seconds"},
		{"bad cron", policyRequest{Name: "x", CronExpr: "not a cron"}, "cron_expr"},
		{"bad sla", policyRequest{Name: "x", SLATargetMinutes: intPtr(-1)}, "SLA/RPO/RTO"},
	}
	for _, tt := range cases {
		if _, err := validatePolicy(&tt.req); err == nil {
			t.Errorf("%s: expected error", tt.name)
		} else if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q must mention %q", tt.name, err.Error(), tt.want)
		}
	}
}

func TestValidatePolicyAccepts(t *testing.T) {
	bTrue, bFalse := true, false
	p, err := validatePolicy(&policyRequest{
		Name: "Hourlies", Strategy: "full", CronExpr: "0 * * * *", Timezone: "Europe/Berlin",
		Enabled: &bTrue, Mode: "compressed", Encrypted: &bTrue, RetentionDays: intPtr(90),
		VerificationEnabled: &bFalse, MaxRetries: intPtr(3), RetryDelaySeconds: intPtr(600),
		SLATargetMinutes: intPtr(60),
	})
	if err != nil {
		t.Fatalf("valid policy rejected: %v", err)
	}
	if p.Strategy != "FULL" || p.Mode != models.ModeCompressed || !p.Encrypted {
		t.Errorf("normalization failed: %+v", p)
	}
	if p.CronExpr != "0 * * * *" || p.Timezone != "Europe/Berlin" {
		t.Errorf("schedule lost: %+v", p)
	}
	if p.MaxRetries != 3 || p.RetryDelaySeconds != 600 || p.SLATargetMinutes != 60 {
		t.Errorf("fields lost: %+v", p)
	}
	if p.VerificationEnabled {
		t.Error("verification_enabled=false must stick")
	}
}

func intPtr(i int) *int { return &i }

// --- EffectivePolicy: policy-first, inline fallback, never half-merged ---

func TestEffectivePolicyInline(t *testing.T) {
	job := &models.BackupJob{
		Mode: models.ModeCompressed, Encrypted: true, RetentionDays: 14,
		Enabled: true, SLATargetMinutes: 30,
		Schedule: &models.BackupSchedule{CronExpr: "0 2 * * *", Timezone: "UTC"},
	}
	eff := models.EffectivePolicy(job)
	if eff.FromPolicy || eff.Mode != models.ModeCompressed || !eff.Encrypted || eff.RetentionDays != 14 {
		t.Errorf("inline resolution wrong: %+v", eff)
	}
	if eff.CronExpr != "0 2 * * *" || !eff.Enabled || eff.MaxRetries != 0 {
		t.Errorf("inline schedule wrong: %+v", eff)
	}
	if !eff.VerificationEnabled {
		t.Error("legacy jobs default to verification on")
	}
}

func TestEffectivePolicyAttached(t *testing.T) {
	pid := uuid.New()
	job := &models.BackupJob{
		PolicyID: &pid,
		Policy: &models.BackupPolicy{
			Mode: models.ModeNormal, Encrypted: false, RetentionDays: 7,
			CronExpr: "*/15 * * * *", Timezone: "America/New_York", Enabled: true,
			MaxRetries: 2, RetryDelaySeconds: 120,
		},
		Mode: models.ModeCompressed, Encrypted: true, RetentionDays: 90,
		Enabled: true,
		Schedule: &models.BackupSchedule{CronExpr: "0 2 * * *", Timezone: "UTC"},
	}
	eff := models.EffectivePolicy(job)
	if !eff.FromPolicy || eff.PolicyID == nil || *eff.PolicyID != pid {
		t.Errorf("must report policy origin: %+v", eff)
	}
	if eff.Mode != models.ModeNormal || eff.Encrypted || eff.RetentionDays != 7 {
		t.Errorf("policy must win wholesale (no half-merge): %+v", eff)
	}
	if eff.CronExpr != "*/15 * * * *" || eff.Timezone != "America/New_York" {
		t.Errorf("policy schedule must win: %+v", eff)
	}
	if eff.MaxRetries != 2 || eff.RetryDelaySeconds != 120 {
		t.Errorf("retry config lost: %+v", eff)
	}
}

func TestEffectivePolicyManualPolicyKeepsJobSchedule(t *testing.T) {
	pid := uuid.New()
	job := &models.BackupJob{
		PolicyID: &pid,
		Policy:   &models.BackupPolicy{Enabled: true, CronExpr: ""},
		Enabled:  true,
		Schedule: &models.BackupSchedule{CronExpr: "0 3 * * *", Timezone: "UTC"},
	}
	eff := models.EffectivePolicy(job)
	if !eff.FromPolicy || eff.CronExpr != "0 3 * * *" || !eff.Enabled {
		t.Errorf("manual policy keeps job schedule: %+v", eff)
	}
}

func TestEffectivePolicyDisabledPolicyWins(t *testing.T) {
	pid := uuid.New()
	job := &models.BackupJob{
		PolicyID: &pid,
		Policy:   &models.BackupPolicy{Enabled: false, CronExpr: ""},
		Enabled:  true,
		Schedule: &models.BackupSchedule{CronExpr: "0 3 * * *", Timezone: "UTC"},
	}
	if eff := models.EffectivePolicy(job); eff.Enabled {
		t.Errorf("disabled policy must disable scheduling: %+v", eff)
	}
}
