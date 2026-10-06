package services

import (
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
)

func retryFixture(t *testing.T, maxRetries, delay int) (*Scheduler, models.BackupJob, models.ResolvedPolicy) {
	t.Helper()
	database := testutil.OpenDB(t)
	user := testutil.CreateUser(t, database, "retry@test.com", "password123")
	org := testutil.CreateOrg(t, database, "Retry Org", user.ID)
	agent := testutil.CreateAgent(t, database, org.ID, "retry-token-"+uuid.New().String())
	target := models.StorageTarget{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "retry-target",
		Type:           models.StorageLocal,
		Path:           t.TempDir(),
	}
	if err := database.Create(&target).Error; err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&target) })
	policy := models.BackupPolicy{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "retry-policy",
		Strategy:       models.StrategyFull,
		Enabled:        true,
		MaxRetries:     maxRetries,
		RetryDelaySeconds: delay,
	}
	if err := database.Create(&policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&policy) })
	job := models.BackupJob{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  org.ID,
		AgentID:         agent.ID,
		StorageTargetID: target.ID,
		PolicyID:        &policy.ID,
		Name:            "retry-job",
		SourceType:      models.SourceFilesystem,
		SourcePath:      t.TempDir(),
		Enabled:         true,
	}
	if err := database.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	t.Cleanup(func() {
		database.Unscoped().Where("backup_job_id = ?", job.ID).Delete(&models.BackupRun{})
		database.Unscoped().Delete(&job)
	})
	database.Preload("Policy").First(&job, job.ID)
	return NewScheduler(database, nil), job, models.EffectivePolicy(&job)
}

func seedFailedRun(t *testing.T, s *Scheduler, job models.BackupJob, attempt int, completedAgo time.Duration) models.BackupRun {
	t.Helper()
	done := time.Now().Add(-completedAgo)
	started := done.Add(-time.Minute)
	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  job.OrganizationID,
		BackupJobID:     job.ID,
		AgentID:         job.AgentID,
		StorageTargetID: job.StorageTargetID,
		Status:          models.RunFailed,
		SourceType:      string(job.SourceType),
		StartedAt:       &started,
		CompletedAt:     &done,
		RetryAttempt:    attempt,
		ErrorMessage:    "boom",
	}
	if err := s.db.Create(&run).Error; err != nil {
		t.Fatalf("seed failed run: %v", err)
	}
	return run
}

func countRuns(t *testing.T, s *Scheduler, jobID uuid.UUID) int64 {
	t.Helper()
	var n int64
	s.db.Model(&models.BackupRun{}).Where("backup_job_id = ?", jobID).Count(&n)
	return n
}

// --- retry engine matrix ---

func TestSchedulerRetryFiresLinkedRun(t *testing.T) {
	s, job, eff := retryFixture(t, 2, 60)
	seedFailedRun(t, s, job, 0, 2*time.Minute)
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 2 {
		t.Fatalf("expected original + 1 retry, got %d runs", n)
	}
	var retry models.BackupRun
	s.db.Where("backup_job_id = ? AND retry_attempt = ?", job.ID, 1).First(&retry)
	if retry.ID == uuid.Nil || retry.Status != models.RunPending {
		t.Fatalf("retry must be a fresh PENDING run, got %+v", retry)
	}
	if retry.RetryOfRunID == nil {
		t.Fatal("retry must link its failed parent")
	}
	if retry.ScheduledFor != nil {
		t.Error("retries are ad-hoc: scheduled_for must stay NULL (idempotency key untouched)")
	}
}

func TestSchedulerRetryRespectsCap(t *testing.T) {
	s, job, eff := retryFixture(t, 2, 60)
	seedFailedRun(t, s, job, 2, 2*time.Minute) // attempts exhausted
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 1 {
		t.Fatalf("exhausted retries must not refire, got %d runs", n)
	}
}

func TestSchedulerRetryHonorsDelay(t *testing.T) {
	s, job, eff := retryFixture(t, 2, 3600)
	seedFailedRun(t, s, job, 0, 2*time.Minute) // younger than the 1h delay
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 1 {
		t.Fatalf("fresh failure must wait out the delay, got %d runs", n)
	}
}

func TestSchedulerRetryDisabledWithoutPolicy(t *testing.T) {
	s, job, eff := retryFixture(t, 0, 60) // max_retries 0
	seedFailedRun(t, s, job, 0, 2*time.Minute)
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 1 {
		t.Fatalf("retries off must not refire, got %d runs", n)
	}
}

func TestSchedulerRetrySkipsInflight(t *testing.T) {
	s, job, eff := retryFixture(t, 3, 60)
	seedFailedRun(t, s, job, 0, 2*time.Minute)
	now := time.Now()
	pending := models.BackupRun{
		Base: models.Base{ID: uuid.New()}, OrganizationID: job.OrganizationID,
		BackupJobID: job.ID, AgentID: job.AgentID, StorageTargetID: job.StorageTargetID,
		Status: models.RunPending, SourceType: string(job.SourceType),
	}
	if err := s.db.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}
	_ = now
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 2 {
		t.Fatalf("inflight work must suppress retry, got %d runs", n)
	}
}

func TestSchedulerRetryYieldsToFreshWork(t *testing.T) {
	s, job, eff := retryFixture(t, 3, 60)
	old := seedFailedRun(t, s, job, 0, 2*time.Hour)
	// A newer successful run exists: the failure is history, not pending work.
	now := time.Now()
	done := now.Add(-time.Hour)
	ok := models.BackupRun{
		Base: models.Base{ID: uuid.New()}, OrganizationID: job.OrganizationID,
		BackupJobID: job.ID, AgentID: job.AgentID, StorageTargetID: job.StorageTargetID,
		Status: models.RunCompleted, SourceType: string(job.SourceType),
		StartedAt: &done, CompletedAt: &done,
	}
	if err := s.db.Create(&ok).Error; err != nil {
		t.Fatal(err)
	}
	_ = old
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 2 {
		t.Fatalf("superseded failure must not retry, got %d runs", n)
	}
}

func TestSchedulerRetryDisabledPolicy(t *testing.T) {
	s, job, _ := retryFixture(t, 3, 60)
	job.Policy.Enabled = false
	eff := models.EffectivePolicy(&job)
	if eff.Enabled {
		t.Fatal("fixture sanity: policy must resolve disabled")
	}
	seedFailedRun(t, s, job, 0, 2*time.Minute)
	if err := s.maybeFire(&job, eff, time.Now().Add(-time.Minute), time.Now()); err != nil {
		t.Fatalf("maybeFire: %v", err)
	}
	if n := countRuns(t, s, job.ID); n != 1 {
		t.Fatalf("disabled policy must never retry, got %d runs", n)
	}
}
