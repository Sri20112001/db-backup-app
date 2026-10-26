package services

import (
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
)

func TestSeverityRank_Ordering(t *testing.T) {
	if models.SeverityRank(models.SeverityCritical) <= models.SeverityRank(models.SeverityWarning) {
		t.Fatal("critical must outrank warning")
	}
	if models.SeverityRank(models.SeverityWarning) <= models.SeverityRank(models.SeverityInfo) {
		t.Fatal("warning must outrank info")
	}
	if models.SeverityRank(models.AlertSeverity("")) != models.SeverityRank(models.SeverityInfo) {
		t.Fatal("unknown severity must default to info rank")
	}
}

func TestDefaultSeverity_PerType(t *testing.T) {
	cases := []struct {
		typ  models.AlertType
		want models.AlertSeverity
	}{
		{models.AlertBackupFailed, models.SeverityCritical},
		{models.AlertBackupMissed, models.SeverityCritical},
		{models.AlertAgentOffline, models.SeverityCritical},
		{models.AlertRestoreFailed, models.SeverityCritical},
		{models.AlertStorageLow, models.SeverityWarning},
		{models.AlertVerifyFailed, models.SeverityWarning},
	}
	for _, c := range cases {
		if got := models.DefaultSeverity(c.typ); got != c.want {
			t.Errorf("DefaultSeverity(%s) = %s, want %s", c.typ, got, c.want)
		}
	}
}

func TestInQuietHours(t *testing.T) {
	noon := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	at23 := time.Date(2026, 6, 15, 23, 30, 0, 0, time.UTC)
	at3 := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		start, end int
		at    time.Time
		want  bool
	}{
		{"disabled start", -1, 6, noon, false},
		{"disabled end", 22, -1, noon, false},
		{"equal ends disabled", 8, 8, time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC), false},
		{"inside normal window", 22, 6, at23, true},
		{"inside normal window early", 22, 6, at3, true},
		{"outside normal window", 22, 6, noon, false},
		{"inside non-wrapping", 1, 5, at3, true},
		{"outside non-wrapping", 1, 5, noon, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := models.NotificationSettings{QuietHoursStart: c.start, QuietHoursEnd: c.end}
			if got := inQuietHours(s, c.at); got != c.want {
				t.Fatalf("inQuietHours(%d-%d, %s) = %v, want %v", c.start, c.end, c.at.Format("15:04"), got, c.want)
			}
		})
	}
}

func TestMissedWindow_Multiplier(t *testing.T) {
	// Hourly cron: interval = 1h. Multiplier scales the window linearly.
	if got := missedWindow("0 * * * *", 2); got != 2*time.Hour {
		t.Fatalf("hourly ×2 = %s, want 2h", got)
	}
	if got := missedWindow("0 * * * *", 3); got != 3*time.Hour {
		t.Fatalf("hourly ×3 = %s, want 3h", got)
	}
	// Daily cron: interval = 24h.
	if got := missedWindow("0 0 * * *", 1); got != 24*time.Hour {
		t.Fatalf("daily ×1 = %s, want 24h", got)
	}
	// Invalid cron falls back to a sane default scaled by the multiplier.
	if got := missedWindow("not-a-cron", 2); got <= 0 {
		t.Fatalf("invalid cron fallback = %s, want positive duration", got)
	}
	// Multiplier below 1 is clamped to 2 so a zero config can't zero the window.
	if got := missedWindow("0 * * * *", 0); got != 2*time.Hour {
		t.Fatalf("multiplier 0 clamped = %s, want 2h", got)
	}
}

// --- Integration (skipped without DATABASE_URL) ---

func TestRaiseAlert_CreatesFiringWithDefaults(t *testing.T) {
	database := testutil.OpenDB(t)
	user := testutil.CreateUser(t, database, "alert-create", "password123")
	org := testutil.CreateOrg(t, database, "Alert Create Org", user.ID)

	alert, created := RaiseAlert(database, nil, AlertInput{
		OrgID:   org.ID,
		Type:    models.AlertBackupFailed,
		Title:   "t",
		Message: "m",
	})
	if !created {
		t.Fatal("first raise must create a row")
	}
	if alert == nil {
		t.Fatal("alert must not be nil when created")
	}
	if alert.Status != models.AlertFiring {
		t.Fatalf("status = %s, want firing", alert.Status)
	}
	if alert.Severity != models.SeverityCritical {
		t.Fatalf("severity = %s, want critical (default for BACKUP_FAILED)", alert.Severity)
	}
	if alert.OccurrenceCount != 1 {
		t.Fatalf("occurrence_count = %d, want 1", alert.OccurrenceCount)
	}
	if alert.LastOccurredAt == nil {
		t.Fatal("last_occurred_at must be set on create")
	}
}

func TestRaiseAlert_CoalescesOnDedupKey(t *testing.T) {
	database := testutil.OpenDB(t)
	user := testutil.CreateUser(t, database, "alert-dedup", "password123")
	org := testutil.CreateOrg(t, database, "Alert Dedup Org", user.ID)

	key := "agent-offline:" + uuid.NewString()
	first, created := RaiseAlert(database, nil, AlertInput{
		OrgID:    org.ID,
		Type:     models.AlertAgentOffline,
		Title:    "Agent offline",
		DedupKey: key,
	})
	if !created {
		t.Fatal("first raise must create")
	}

	second, created2 := RaiseAlert(database, nil, AlertInput{
		OrgID:    org.ID,
		Type:     models.AlertAgentOffline,
		Title:    "Agent offline",
		Message:  "still down",
		DedupKey: key,
	})
	if created2 {
		t.Fatal("second raise with same dedup key must coalesce, not create")
	}
	if second.ID != first.ID {
		t.Fatalf("coalesced alert id = %s, want %s", second.ID, first.ID)
	}
	if second.OccurrenceCount != 2 {
		t.Fatalf("occurrence_count = %d, want 2", second.OccurrenceCount)
	}

	var count int64
	database.Model(&models.Alert{}).
		Where("organization_id = ? AND dedup_key = ?", org.ID, key).
		Count(&count)
	if count != 1 {
		t.Fatalf("row count = %d, want 1 (partial unique index on firing)", count)
	}
}

func TestResolveAlerts_ThenNewRaiseMakesNewRow(t *testing.T) {
	database := testutil.OpenDB(t)
	user := testutil.CreateUser(t, database, "alert-resolve", "password123")
	org := testutil.CreateOrg(t, database, "Alert Resolve Org", user.ID)

	key := "backup-missed:" + uuid.NewString()
	first, _ := RaiseAlert(database, nil, AlertInput{
		OrgID:    org.ID,
		Type:     models.AlertBackupMissed,
		Title:    "missed",
		DedupKey: key,
	})

	if n := ResolveAlerts(database, org.ID, models.AlertBackupMissed, key); n != 1 {
		t.Fatalf("resolved %d, want 1", n)
	}

	var reloaded models.Alert
	database.First(&reloaded, "id = ?", first.ID)
	if reloaded.Status != models.AlertResolved {
		t.Fatalf("status after resolve = %s, want resolved", reloaded.Status)
	}
	if reloaded.ResolvedAt == nil {
		t.Fatal("resolved_at must be set")
	}

	// Resolving again is a no-op (only FIRING rows match).
	if n := ResolveAlerts(database, org.ID, models.AlertBackupMissed, key); n != 0 {
		t.Fatalf("second resolve = %d, want 0", n)
	}

	// A fresh raise after resolve creates a NEW row (the unique index is
	// partial on firing only), so the flap can re-alert.
	again, created := RaiseAlert(database, nil, AlertInput{
		OrgID:    org.ID,
		Type:     models.AlertBackupMissed,
		Title:    "missed again",
		DedupKey: key,
	})
	if !created {
		t.Fatal("raise after resolve must create a new row")
	}
	if again.ID == first.ID {
		t.Fatal("new raise must not reuse the resolved row")
	}
}
