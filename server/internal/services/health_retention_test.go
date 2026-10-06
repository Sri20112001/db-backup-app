package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
)

func retentionFixture(t *testing.T) (*HealthMonitor, models.BackupJob, string) {
	t.Helper()
	database := testutil.OpenDB(t)
	user := testutil.CreateUser(t, database, "retention@test.com", "password123")
	org := testutil.CreateOrg(t, database, "Retention Org", user.ID)
	agent := testutil.CreateAgent(t, database, org.ID, "retention-token-"+uuid.New().String())
	dir := t.TempDir()
	target := models.StorageTarget{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "retention-target",
		Type:           models.StorageLocal,
		Path:           dir,
	}
	if err := database.Create(&target).Error; err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&target) })
	job := models.BackupJob{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  org.ID,
		AgentID:         agent.ID,
		StorageTargetID: target.ID,
		Name:            "retention-job",
		SourceType:      models.SourceFilesystem,
		SourcePath:      dir,
		Enabled:         true,
		RetentionDays:   30,
	}
	if err := database.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	t.Cleanup(func() {
		database.Unscoped().Where("backup_job_id = ?", job.ID).Delete(&models.BackupRun{})
		database.Unscoped().Where("backup_job_id = ?", job.ID).Delete(&models.BackupArtifact{})
		database.Unscoped().Delete(&job)
	})
	return &HealthMonitor{db: database}, job, dir
}

func seedAgedRun(t *testing.T, h *HealthMonitor, job models.BackupJob, status models.BackupRunStatus, age time.Duration, withArtifact bool) (models.BackupRun, string) {
	t.Helper()
	done := time.Now().Add(-age)
	started := done.Add(-time.Minute)
	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  job.OrganizationID,
		BackupJobID:     job.ID,
		AgentID:         job.AgentID,
		StorageTargetID: job.StorageTargetID,
		Status:          status,
		SourceType:      string(job.SourceType),
		StartedAt:       &started,
		CompletedAt:     &done,
		StoragePath:     filepath.Join(job.SourcePath, "run-"+runIDShort()+".tar"),
	}
	if err := h.db.Create(&run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if withArtifact {
		content := []byte("payload-for-" + run.ID.String())
		if err := os.WriteFile(run.StoragePath, content, 0600); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		if err := h.db.Create(&models.BackupArtifact{
			Base: models.Base{ID: uuid.New()}, BackupRunID: run.ID,
			Name: "a.tar", Size: int64(len(content)), StoragePath: run.StoragePath,
		}).Error; err != nil {
			t.Fatalf("seed artifact: %v", err)
		}
	}
	return run, run.StoragePath
}

func runIDShort() string { return uuid.New().String()[:8] }

func runExists(t *testing.T, h *HealthMonitor, id uuid.UUID) bool {
	t.Helper()
	var n int64
	h.db.Model(&models.BackupRun{}).Where("id = ?", id).Count(&n)
	return n == 1
}

// Expired FAILED and CANCELLED runs are not recovery points: retention must
// remove them (and their files) on the same schedule as old COMPLETED runs.
func TestRetentionCleansDeadRuns(t *testing.T) {
	h, job, _ := retentionFixture(t)
	old := 31 * 24 * time.Hour

	failed, failedPath := seedAgedRun(t, h, job, models.RunFailed, old, true)
	cancelled, _ := seedAgedRun(t, h, job, models.RunCancelled, old, true)
	completed, completedPath := seedAgedRun(t, h, job, models.RunCompleted, old, true)
	freshFailed, _ := seedAgedRun(t, h, job, models.RunFailed, time.Hour, false)

	h.enforceRetention()

	if runExists(t, h, failed.ID) {
		t.Error("expired FAILED run must be deleted")
	}
	if runExists(t, h, cancelled.ID) {
		t.Error("expired CANCELLED run must be deleted")
	}
	if runExists(t, h, completed.ID) {
		t.Error("expired COMPLETED run must be deleted")
	}
	if !runExists(t, h, freshFailed.ID) {
		t.Error("fresh FAILED run must survive (below retention cutoff)")
	}
	for _, p := range []string{failedPath, completedPath} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expired file must be removed from LOCAL target: %s", p)
		}
	}
	var chunks int64
	h.db.Model(&models.BackupChunk{}).Count(&chunks)
	if chunks != 0 {
		t.Errorf("expected no chunks, got %d", chunks)
	}
}

// Retention is status-aware: in-flight and recent runs of every status survive.
func TestRetentionKeepsLiveRuns(t *testing.T) {
	h, job, _ := retentionFixture(t)

	running, _ := seedAgedRun(t, h, job, models.RunRunning, 0, false)
	h.db.Model(&running).Update("completed_at", nil)
	recentCompleted, _ := seedAgedRun(t, h, job, models.RunCompleted, time.Hour, false)

	h.enforceRetention()

	if !runExists(t, h, running.ID) {
		t.Error("in-flight RUNNING run must survive retention")
	}
	if !runExists(t, h, recentCompleted.ID) {
		t.Error("fresh COMPLETED run must survive retention")
	}
}
