// Package handlers_test — reliability audit regressions. Every test here
// proves a failure/race/tenant-safety property of the backup lifecycle over
// HTTP against a real database (skips without DATABASE_URL).
package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
)

// auditFixture extends the enrollment fixture with storage, agent (known raw
// token), and helpers to mint jobs/runs directly.
type auditFixture struct {
	*enrollFixture
	agent   models.Agent
	rawTok  string
	storage models.StorageTarget
}

func setupAuditFixture(t *testing.T, orgName, email string) *auditFixture {
	t.Helper()
	f := setupEnrollFixture(t, orgName, email)
	rawTok := "audit-token-" + uuid.New().String()
	agent := testutil.CreateAgent(t, f.db, f.org.ID, rawTok)
	// testutil agent defaults ONLINE without last_seen; pin recency so the
	// agent reads alive unless a test deliberately ages it.
	now := time.Now()
	f.db.Model(&agent).Updates(map[string]interface{}{"last_seen_at": &now})
	storage := models.StorageTarget{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: f.org.ID,
		Name:           "audit-target",
		Type:           models.StorageLocal,
		Path:           t.TempDir(),
	}
	if err := f.db.Create(&storage).Error; err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { f.db.Unscoped().Delete(&storage) })
	return &auditFixture{enrollFixture: f, agent: agent, rawTok: rawTok, storage: storage}
}

func (f *auditFixture) agentHeaders() map[string]string {
	return agentAuth(f.agent.ID.String(), f.rawTok)
}

func (f *auditFixture) createJob(t *testing.T, extra map[string]interface{}) string {
	t.Helper()
	body := map[string]interface{}{
		"agent_id": f.agent.ID.String(), "storage_target_id": f.storage.ID.String(),
		"name": "audit-job-" + uuid.New().String()[:8],
		"source_type": "FILESYSTEM", "source_path": t.TempDir(),
	}
	for k, v := range extra {
		body[k] = v
	}
	rec := doReq(t, f.router, "POST",
		"/vaultguard/api/organizations/"+f.org.ID.String()+"/backup-jobs",
		body, userAuth(f.jwt))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create job: %d %s", rec.Code, rec.Body.String())
	}
	id, _ := decodeBody(t, rec)["id"].(string)
	t.Cleanup(func() {
		f.db.Unscoped().Where("backup_job_id = ?", id).Delete(&models.BackupRun{})
		f.db.Unscoped().Where("id = ?", id).Delete(&models.BackupJob{})
	})
	return id
}

func (f *auditFixture) seedRun(t *testing.T, jobID string, status models.BackupRunStatus) models.BackupRun {
	t.Helper()
	jid, _ := uuid.Parse(jobID)
	now := time.Now()
	run := models.BackupRun{
		Base: models.Base{ID: uuid.New()}, OrganizationID: f.org.ID,
		BackupJobID: jid, AgentID: f.agent.ID, StorageTargetID: f.storage.ID,
		Status: status, SourceType: "FILESYSTEM", StartedAt: &now,
	}
	if err := f.db.Create(&run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	t.Cleanup(func() {
		f.db.Unscoped().Where("backup_run_id = ?", run.ID).Delete(&models.BackupArtifact{})
		f.db.Unscoped().Where("id = ?", run.ID).Delete(&models.BackupRun{})
	})
	return run
}

// --- duplicate claim: exactly one winner, loser gets 409, stranger gets 404 ---

func TestAudit_DuplicateClaim(t *testing.T) {
	f := setupAuditFixture(t, "Audit Claim Org", "audit-claim@test.com")
	jobID := f.createJob(t, nil)
	run := f.seedRun(t, jobID, models.RunPending)
	path := "/vaultguard/api/agent/runs/" + run.ID.String() + "/claim"

	rec := doReq(t, f.router, "POST", path, nil, f.agentHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("first claim: %d %s", rec.Code, rec.Body.String())
	}
	rec = doReq(t, f.router, "POST", path, nil, f.agentHeaders())
	if rec.Code != http.StatusConflict {
		t.Errorf("second claim: %d, want 409", rec.Code)
	}

	// Another org's agent must not even see the run (404, not 409).
	other := setupAuditFixture(t, "Audit Claim Org B", "audit-claim-b@test.com")
	rec = doReq(t, other.router, "POST", path, nil, other.agentHeaders())
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-agent claim: %d, want 404", rec.Code)
	}
}

// --- cancel idempotency: double cancel succeeds ---

func TestAudit_CancelIdempotent(t *testing.T) {
	f := setupAuditFixture(t, "Audit Cancel Org", "audit-cancel@test.com")
	jobID := f.createJob(t, nil)
	run := f.seedRun(t, jobID, models.RunPending)
	path := "/vaultguard/api/organizations/" + f.org.ID.String() + "/backup-runs/" + run.ID.String() + "/cancel"
	for i := 0; i < 2; i++ {
		if rec := doReq(t, f.router, "POST", path, nil, userAuth(f.jwt)); rec.Code != http.StatusOK {
			t.Fatalf("cancel #%d: %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
}

// --- RunNow guard: in-flight run blocks a second manual run ---

func TestAudit_RunNowInflightConflict(t *testing.T) {
	f := setupAuditFixture(t, "Audit RunNow Org", "audit-runnow@test.com")
	jobID := f.createJob(t, nil)
	path := "/vaultguard/api/organizations/" + f.org.ID.String() + "/backup-jobs/" + jobID + "/run"
	if rec := doReq(t, f.router, "POST", path, nil, userAuth(f.jwt)); rec.Code != http.StatusCreated {
		t.Fatalf("first run: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(t, f.router, "POST", path, nil, userAuth(f.jwt)); rec.Code != http.StatusConflict {
		t.Errorf("second run while inflight: %d, want 409", rec.Code)
	}
}

// --- restore duplicates collapse; foreign agents rejected ---

func TestAudit_RestoreDuplicateAndOrgCheck(t *testing.T) {
	f := setupAuditFixture(t, "Audit Restore Org", "audit-restore@test.com")
	jobID := f.createJob(t, nil)
	run := f.seedRun(t, jobID, models.RunCompleted)
	body := map[string]interface{}{
		"backup_run_id": run.ID.String(), "agent_id": f.agent.ID.String(),
		"destination_path": t.TempDir(), "confirmed": true,
	}
	path := "/vaultguard/api/organizations/" + f.org.ID.String() + "/restores"
	first := doReq(t, f.router, "POST", path, body, userAuth(f.jwt))
	if first.Code != http.StatusCreated {
		t.Fatalf("first restore: %d %s", first.Code, first.Body.String())
	}
	firstID := decodeBody(t, first)["id"].(string)
	second := doReq(t, f.router, "POST", path, body, userAuth(f.jwt))
	if second.Code != http.StatusOK {
		t.Fatalf("duplicate restore: %d, want idempotent 200", second.Code)
	}
	if decodeBody(t, second)["id"] != firstID {
		t.Error("duplicate restore must return the live job, not a fork")
	}

	other := setupAuditFixture(t, "Audit Restore Org B", "audit-restore-b@test.com")
	body["agent_id"] = other.agent.ID.String()
	if rec := doReq(t, f.router, "POST", path, body, userAuth(f.jwt)); rec.Code != http.StatusNotFound {
		t.Errorf("foreign agent restore: %d, want 404", rec.Code)
	}
}

// --- artifact duplicates collapse to one row ---

func TestAudit_ArtifactDuplicate(t *testing.T) {
	f := setupAuditFixture(t, "Audit Artifact Org", "audit-artifact@test.com")
	jobID := f.createJob(t, nil)
	run := f.seedRun(t, jobID, models.RunRunning)
	body := map[string]interface{}{
		"name": "a.tar", "size": 10, "checksum": "abc", "storage_path": "/tmp/a.tar",
	}
	path := "/vaultguard/api/backup-runs/" + run.ID.String() + "/artifacts"
	first := doReq(t, f.router, "POST", path, body, f.agentHeaders())
	if first.Code != http.StatusCreated {
		t.Fatalf("first artifact: %d %s", first.Code, first.Body.String())
	}
	second := doReq(t, f.router, "POST", path, body, f.agentHeaders())
	if second.Code != http.StatusOK {
		t.Fatalf("duplicate artifact: %d, want idempotent 200", second.Code)
	}
	if decodeBody(t, first)["id"] != decodeBody(t, second)["id"] {
		t.Error("duplicate artifact must return the existing row")
	}
	var n int64
	f.db.Model(&models.BackupArtifact{}).Where("backup_run_id = ?", run.ID).Count(&n)
	if n != 1 {
		t.Errorf("expected exactly 1 artifact row, got %d", n)
	}
}

// --- concurrent terminal reports: exactly one wins ---

func TestAudit_ConcurrentTerminalReports(t *testing.T) {
	f := setupAuditFixture(t, "Audit Race Org", "audit-race@test.com")
	jobID := f.createJob(t, nil)
	run := f.seedRun(t, jobID, models.RunVerifying)
	path := "/vaultguard/api/backup-runs/" + run.ID.String() + "/status"

	results := make([]int, 2)
	done := make(chan struct{}, 2)
	go func() {
		results[0] = doReq(t, f.router, "PUT", path,
			map[string]interface{}{"status": "COMPLETED", "checksum": "x"}, f.agentHeaders()).Code
		done <- struct{}{}
	}()
	go func() {
		results[1] = doReq(t, f.router, "PUT", path,
			map[string]interface{}{"status": "FAILED", "error_message": "boom"}, f.agentHeaders()).Code
		done <- struct{}{}
	}()
	<-done
	<-done
	ok := 0
	for _, c := range results {
		if c == http.StatusOK {
			ok++
		} else if c != http.StatusConflict && c != http.StatusBadRequest {
			t.Errorf("unexpected code %d", c)
		}
	}
	if ok != 1 {
		t.Errorf("exactly one terminal report must win, got %v", results)
	}
	var fresh models.BackupRun
	f.db.Where("id = ?", run.ID).First(&fresh)
	if fresh.Status != models.RunCompleted && fresh.Status != models.RunFailed {
		t.Errorf("run must land terminal, got %s", fresh.Status)
	}
}

// --- dashboard health honors effective policy ---

func TestAudit_DashboardPolicyHealth(t *testing.T) {
	f := setupAuditFixture(t, "Audit Health Org", "audit-health@test.com")
	orgPath := "/vaultguard/api/organizations/" + f.org.ID.String()

	rec := doReq(t, f.router, "POST", orgPath+"/backup-policies", map[string]interface{}{
		"name": "P", "cron_expr": "* * * * *", "enabled": false,
	}, userAuth(f.jwt))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create policy: %d %s", rec.Code, rec.Body.String())
	}
	pid := decodeBody(t, rec)["id"].(string)
	f.createJob(t, map[string]interface{}{"policy_id": pid})

	rec = doReq(t, f.router, "GET", orgPath+"/dashboard/health", nil, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("health: %d", rec.Code)
	}
	if rec.Body.String() != "[]" && rec.Body.String() != "null" {
		t.Errorf("disabled-policy job must be excluded from health, got %s", rec.Body.String())
	}
}
