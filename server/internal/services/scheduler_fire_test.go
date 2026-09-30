package services

import (
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
)

// TestSchedulerFireIdempotent proves two racing ticks for the same cron
// boundary produce exactly one run (UNIQUE(backup_job_id, scheduled_for)
// + ON CONFLICT DO NOTHING). Needs DATABASE_URL; skipped otherwise, like
// the rest of the DB-gated suite.
func TestSchedulerFireIdempotent(t *testing.T) {
	database := testutil.OpenDB(t)

	user := testutil.CreateUser(t, database, "sched-test@test.com", "password123")
	org := testutil.CreateOrg(t, database, "Sched Org", user.ID)
	agent := testutil.CreateAgent(t, database, org.ID, "sched-token-"+uuid.New().String())

	target := models.StorageTarget{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "sched-target",
		Type:           models.StorageLocal,
		Path:           t.TempDir(),
	}
	if err := database.Create(&target).Error; err != nil {
		t.Fatalf("create storage target: %v", err)
	}

	job := models.BackupJob{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  org.ID,
		AgentID:         agent.ID,
		StorageTargetID: target.ID,
		Name:            "sched-job",
		SourceType:      models.SourceFilesystem,
		SourcePath:      t.TempDir(),
		Enabled:         true,
	}
	if err := database.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	s := NewScheduler(database, nil)
	boundary := time.Now().UTC().Truncate(time.Second)

	// Two ticks, same boundary — the second must be a silent no-op.
	s.fire(&job, boundary)
	s.fire(&job, boundary)

	var count int64
	database.Model(&models.BackupRun{}).
		Where("backup_job_id = ?", job.ID).
		Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 run after duplicate fire, got %d", count)
	}

	// A different boundary still fires normally (once the first run leaves
	// the in-flight states — the backlog guard, not the idempotency guard).
	database.Model(&models.BackupRun{}).
		Where("backup_job_id = ?", job.ID).
		Update("status", models.RunCompleted)
	s.fire(&job, boundary.Add(time.Minute))
	database.Model(&models.BackupRun{}).
		Where("backup_job_id = ?", job.ID).
		Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 runs after distinct boundary, got %d", count)
	}

	t.Cleanup(func() {
		database.Unscoped().Where("backup_job_id = ?", job.ID).Delete(&models.BackupRun{})
		database.Unscoped().Delete(&job)
		database.Unscoped().Delete(&target)
	})
}
