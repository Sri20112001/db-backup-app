package services

import (
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/google/uuid"
)

func testJob(cron, tz string) *models.BackupJob {
	return &models.BackupJob{
		Base: models.Base{ID: uuid.New()},
		Schedule: &models.BackupSchedule{
			CronExpr: cron,
			Timezone: tz,
		},
	}
}

func TestSchedulerDue(t *testing.T) {
	s := NewScheduler(nil, nil)

	// Every-minute cron: a 70s window always contains a firing boundary.
	job := testJob("*/1 * * * *", "UTC")
	now := time.Now().UTC().Truncate(time.Second)
	if !s.due(job, now.Add(-70*time.Second), now) {
		t.Error("expected due within 70s window for */1 * * * *")
	}

	// 10s window ending exactly on a minute boundary may or may not contain
	// one; a 5s window inside a minute never does.
	midMinute := time.Date(2026, 9, 25, 10, 30, 30, 0, time.UTC)
	if s.due(job, midMinute.Add(-5*time.Second), midMinute) {
		t.Error("expected not-due inside a minute for */1 * * * *")
	}

	// Daily 23:00 UTC: window crossing it fires, other windows don't.
	daily := testJob("0 23 * * *", "UTC")
	crossing := time.Date(2026, 9, 25, 23, 0, 30, 0, time.UTC)
	if !s.due(daily, crossing.Add(-time.Minute), crossing) {
		t.Error("expected due crossing 23:00 for daily job")
	}
	quiet := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	if s.due(daily, quiet.Add(-time.Minute), quiet) {
		t.Error("expected not-due at 10:00 for daily job")
	}

	// Invalid cron never fires (and must not panic).
	bad := testJob("not a cron", "UTC")
	if s.due(bad, quiet.Add(-time.Hour), quiet) {
		t.Error("expected not-due for invalid cron")
	}
}
