// Package handlers_test exercises backup policy, restore gating, run
// verification transitions, and idempotency over HTTP against a real
// database. DB-gated via testutil.OpenDB (skips without DATABASE_URL).
package handlers_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
)

type policyFixture struct {
	*enrollFixture
	agent   models.Agent
	rawTok  string
	storage models.StorageTarget
}

func setupPolicyFixture(t *testing.T, orgName, email string) *policyFixture {
	t.Helper()
	f := setupEnrollFixture(t, orgName, email)
	rawTok := "agent-token-" + uuid.New().String()
	agent := testutil.CreateAgent(t, f.db, f.org.ID, rawTok)
	storage := models.StorageTarget{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: f.org.ID,
		Name:           "local-" + uuid.New().String()[:8],
		Type:           models.StorageLocal,
		Path:           t.TempDir(),
	}
	if err := f.db.Create(&storage).Error; err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { f.db.Unscoped().Delete(&storage) })
	return &policyFixture{enrollFixture: f, agent: agent, rawTok: rawTok, storage: storage}
}

func (f *policyFixture) createJob(t *testing.T, extra map[string]interface{}) string {
	t.Helper()
	body := map[string]interface{}{
		"agent_id":          f.agent.ID.String(),
		"storage_target_id": f.storage.ID.String(),
		"name":              "job-" + uuid.New().String()[:8],
		"source_type":       "FILESYSTEM",
		"source_path":       t.TempDir(),
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
	return decodeBody(t, rec)["id"].(string)
}

func (f *policyFixture) updateJob(t *testing.T, jobID string, extra map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]interface{}{
		"agent_id":          f.agent.ID.String(),
		"storage_target_id": f.storage.ID.String(),
		"name":              "j",
		"source_type":       "FILESYSTEM",
		"source_path":       t.TempDir(),
	}
	for k, v := range extra {
		body[k] = v
	}
	return doReq(t, f.router, "PUT",
		"/vaultguard/api/organizations/"+f.org.ID.String()+"/backup-jobs/"+jobID,
		body, userAuth(f.jwt))
}

func (f *policyFixture) createRun(t *testing.T, jobID string, status models.BackupRunStatus, completedAgo time.Duration) models.BackupRun {
	t.Helper()
	jid, _ := uuid.Parse(jobID)
	now := time.Now()
	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  f.org.ID,
		BackupJobID:     jid,
		AgentID:         f.agent.ID,
		StorageTargetID: f.storage.ID,
		Status:          status,
		SourceType:      "FILESYSTEM",
		StartedAt:       &now,
	}
	if status == models.RunCompleted || status == models.RunFailed || status == models.RunCancelled {
		done := now.Add(-completedAgo)
		run.CompletedAt = &done
	}
	if err := f.db.Create(&run).Error; err != nil {
		t.Fatalf("create run: %v", err)
	}
	t.Cleanup(func() { f.db.Unscoped().Delete(&run) })
	return run
}

// --- policy CRUD + validation + isolation ---

func TestPolicyFlow_CRUD(t *testing.T) {
	f := setupPolicyFixture(t, "Policy Org", "policy@test.com")
	base := "/vaultguard/api/organizations/" + f.org.ID.String() + "/backup-policies"

	rec := doReq(t, f.router, "POST", base, map[string]interface{}{
		"name": "Nightly", "cron_expr": "0 2 * * *", "retention_days": 14,
		"max_retries": 2, "retry_delay_seconds": 120,
	}, userAuth(f.jwt))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	created := decodeBody(t, rec)
	id, _ := created["id"].(string)
	if id == "" || created["strategy"] != "FULL" || created["timezone"] != "UTC" {
		t.Fatalf("bad create response: %s", rec.Body.String())
	}

	rec = doReq(t, f.router, "GET", base+"/"+id, nil, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
	if decodeBody(t, rec)["attached_jobs"].(float64) != 0 {
		t.Fatalf("fresh policy must have 0 jobs: %s", rec.Body.String())
	}

	rec = doReq(t, f.router, "PUT", base+"/"+id, map[string]interface{}{
		"name": "Nightly", "retention_days": 30, "enabled": false,
	}, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}
	if decodeBody(t, rec)["enabled"] != false {
		t.Fatalf("update lost fields: %s", rec.Body.String())
	}

	rec = doReq(t, f.router, "DELETE", base+"/"+id, nil, userAuth(f.jwt))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := doReq(t, f.router, "GET", base+"/"+id, nil, userAuth(f.jwt)); rec.Code != http.StatusNotFound {
		t.Fatalf("deleted policy still visible: %d", rec.Code)
	}
}

func TestPolicyFlow_Validation(t *testing.T) {
	f := setupPolicyFixture(t, "PolicyVal Org", "policyval@test.com")
	base := "/vaultguard/api/organizations/" + f.org.ID.String() + "/backup-policies"
	for name, body := range map[string]map[string]interface{}{
		"empty":      {},
		"incremental": {"name": "x", "strategy": "INCREMENTAL"},
		"bad cron":   {"name": "x", "cron_expr": "never o'clock"},
		"neg retain": {"name": "x", "retention_days": -1},
		"many retry": {"name": "x", "max_retries": 99},
	} {
		rec := doReq(t, f.router, "POST", base, body, userAuth(f.jwt))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
	}
}

func TestPolicyFlow_Isolation(t *testing.T) {
	fa := setupPolicyFixture(t, "Pol Iso A", "poliso-a@test.com")
	fb := setupPolicyFixture(t, "Pol Iso B", "poliso-b@test.com")
	baseA := "/vaultguard/api/organizations/" + fa.org.ID.String() + "/backup-policies"
	rec := doReq(t, fa.router, "POST", baseA, map[string]interface{}{"name": "A-policy"}, userAuth(fa.jwt))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d", rec.Code)
	}
	pid := decodeBody(t, rec)["id"].(string)

	// Org B (separate router+DB rows, same shared test DB) must 404 everywhere.
	baseB := "/vaultguard/api/organizations/" + fb.org.ID.String() + "/backup-policies"
	if r := doReq(t, fb.router, "GET", baseB+"/"+pid, nil, userAuth(fb.jwt)); r.Code != http.StatusNotFound {
		t.Errorf("cross-org get: %d, want 404", r.Code)
	}
	if r := doReq(t, fb.router, "PUT", baseB+"/"+pid, map[string]interface{}{"name": "hijack"}, nil); r.Code == http.StatusOK {
		t.Errorf("cross-org update without auth must not succeed: %d", r.Code)
	}
	// Authenticated as B but addressing A's policy id under B's org path.
	if r := doReq(t, fb.router, "PUT", baseB+"/"+pid, map[string]interface{}{"name": "hijack"}, userAuth(fb.jwt)); r.Code != http.StatusNotFound {
		t.Errorf("cross-org update: %d, want 404", r.Code)
	}
	if r := doReq(t, fb.router, "DELETE", baseB+"/"+pid, nil, userAuth(fb.jwt)); r.Code != http.StatusNotFound {
		t.Errorf("cross-org delete: %d, want 404", r.Code)
	}
	if r := doReq(t, fb.router, "GET", baseB, nil, userAuth(fb.jwt)); r.Code != http.StatusOK {
		t.Fatalf("list: %d", r.Code)
	} else if r.Body.String() != "[]" {
		t.Errorf("org B list must be empty, got %s", r.Body.String())
	}
}

// --- attach/detach lifecycle ---

func TestPolicyFlow_AttachDetach(t *testing.T) {
	f := setupPolicyFixture(t, "Attach Org", "attach@test.com")
	orgPath := "/vaultguard/api/organizations/" + f.org.ID.String()

	rec := doReq(t, f.router, "POST", orgPath+"/backup-policies",
		map[string]interface{}{"name": "P", "mode": "COMPRESSED"}, userAuth(f.jwt))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create policy: %d", rec.Code)
	}
	pid := decodeBody(t, rec)["id"].(string)

	jobID := f.createJob(t, nil)
	rec = doReq(t, f.router, "GET", orgPath+"/backup-jobs/"+jobID, nil, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("get job: %d", rec.Code)
	}

	// Unknown policy rejected.
	rec = f.updateJob(t, jobID, map[string]interface{}{"policy_id": uuid.New().String()})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown policy attach: %d, want 400", rec.Code)
	}

	// Attach.
	rec = f.updateJob(t, jobID, map[string]interface{}{"policy_id": pid})
	if rec.Code != http.StatusOK {
		t.Fatalf("attach: %d %s", rec.Code, rec.Body.String())
	}
	rec = doReq(t, f.router, "GET", orgPath+"/backup-jobs/"+jobID, nil, userAuth(f.jwt))
	job := decodeBody(t, rec)
	if job["policy_id"] != pid {
		t.Fatalf("job missing policy link: %s", rec.Body.String())
	}
	if job["policy"] == nil {
		t.Fatalf("job must preload policy: %s", rec.Body.String())
	}

	// Delete detaches (job survives on inline settings).
	rec = doReq(t, f.router, "DELETE", orgPath+"/backup-policies/"+pid, nil, userAuth(f.jwt))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete policy: %d", rec.Code)
	}
	rec = doReq(t, f.router, "GET", orgPath+"/backup-jobs/"+jobID, nil, userAuth(f.jwt))
	if _, has := decodeBody(t, rec)["policy_id"]; has {
		t.Fatalf("deleted policy must detach, got: %s", rec.Body.String())
	}
}

// --- restore gate: only COMPLETED runs are candidates ---

func TestRestoreFlow_Gate(t *testing.T) {
	f := setupPolicyFixture(t, "Restore Org", "restore-gate@test.com")
	orgPath := "/vaultguard/api/organizations/" + f.org.ID.String()
	jobID := f.createJob(t, nil)

	tryRestore := func(runID string) int {
		rec := doReq(t, f.router, "POST", orgPath+"/restores", map[string]interface{}{
			"backup_run_id": runID, "agent_id": f.agent.ID.String(),
			"destination_path": t.TempDir(), "confirmed": true,
		}, userAuth(f.jwt))
		return rec.Code
	}
	if code := tryRestore(f.createRun(t, jobID, models.RunFailed, time.Hour).ID.String()); code != http.StatusBadRequest {
		t.Errorf("failed run restore: %d, want 400", code)
	}
	if code := tryRestore(f.createRun(t, jobID, models.RunRunning, 0).ID.String()); code != http.StatusBadRequest {
		t.Errorf("running run restore: %d, want 400", code)
	}
	if code := tryRestore(f.createRun(t, jobID, models.RunCompleted, time.Hour).ID.String()); code != http.StatusCreated {
		t.Errorf("completed run restore: %d, want 201", code)
	}
}

// --- verification transitions + idempotency ---

func TestRunFlow_VerifiedTransitions(t *testing.T) {
	f := setupPolicyFixture(t, "Verify Org", "verify-flow@test.com")
	ah := agentAuth(f.agent.ID.String(), f.rawTok)
	jobID := f.createJob(t, nil)
	run := f.createRun(t, jobID, models.RunRunning, 0)
	f.db.Create(&models.BackupArtifact{
		Base: models.Base{ID: uuid.New()}, BackupRunID: run.ID,
		Name: "a.tar", Size: 10, Checksum: "abc", StoragePath: "/tmp/a.tar",
	})
	statusPath := "/vaultguard/api/backup-runs/" + run.ID.String() + "/status"
	advance := func(status string, extra map[string]interface{}) *httptest.ResponseRecorder {
		t.Helper()
		body := map[string]interface{}{"status": status}
		for k, v := range extra {
			body[k] = v
		}
		return doReq(t, f.router, "PUT", statusPath, body, ah)
	}

	// Walk the real agent stage chain; COMPLETED marks PENDING artifacts
	// VERIFIED (agent HEAD-verified first).
	for _, s := range []string{"UPLOADING", "VERIFYING"} {
		if rec := advance(s, nil); rec.Code != http.StatusOK {
			t.Fatalf("stage %s: %d %s", s, rec.Code, rec.Body.String())
		}
	}
	rec := advance("COMPLETED", map[string]interface{}{"checksum": "abc"})
	if rec.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body.String())
	}
	var art models.BackupArtifact
	f.db.Where("backup_run_id = ?", run.ID).First(&art)
	if art.VerificationStatus != models.ArtifactVerified || art.VerifiedAt == nil {
		t.Errorf("artifact must be VERIFIED with timestamp, got %+v", art)
	}

	// Duplicate COMPLETED report: idempotent success, not an error.
	rec = doReq(t, f.router, "PUT", statusPath, map[string]interface{}{
		"status": "COMPLETED", "checksum": "abc",
	}, ah)
	if rec.Code != http.StatusOK {
		t.Errorf("duplicate COMPLETED: %d, want idempotent 200", rec.Code)
	}

	// Illegal backward transition rejected.
	rec = doReq(t, f.router, "PUT", statusPath, map[string]interface{}{"status": "RUNNING"}, ah)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("COMPLETED->RUNNING: %d, want 400", rec.Code)
	}
}

func TestRunFlow_VerifyEndpoint(t *testing.T) {
	f := setupPolicyFixture(t, "VerifyEp Org", "verifyep@test.com")
	orgPath := "/vaultguard/api/organizations/" + f.org.ID.String()
	content := []byte("vaultguard-verify-content")
	p := filepath.Join(f.storage.Path, "run.bin")
	if err := os.WriteFile(p, content, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	run := f.createRun(t, f.createJob(t, nil), models.RunCompleted, time.Hour)
	f.db.Model(&run).Updates(map[string]interface{}{"checksum": hex.EncodeToString(sum[:]), "storage_path": p})
	f.db.Create(&models.BackupArtifact{
		Base: models.Base{ID: uuid.New()}, BackupRunID: run.ID,
		Name: "run.bin", Size: int64(len(content)), StoragePath: p,
	})

	rec := doReq(t, f.router, "POST", orgPath+"/backup-runs/"+run.ID.String()+"/verify", nil, userAuth(f.jwt))
	if rec.Code != http.StatusOK || decodeBody(t, rec)["match"] != true {
		t.Fatalf("verify match: %d %s", rec.Code, rec.Body.String())
	}
	var art models.BackupArtifact
	f.db.Where("backup_run_id = ?", run.ID).First(&art)
	if art.VerificationStatus != models.ArtifactVerified {
		t.Errorf("verify endpoint must persist VERIFIED, got %s", art.VerificationStatus)
	}

	// Tamper → mismatch + FAILED persisted.
	if err := os.WriteFile(p, []byte("tampered!!"), 0600); err != nil {
		t.Fatal(err)
	}
	rec = doReq(t, f.router, "POST", orgPath+"/backup-runs/"+run.ID.String()+"/verify", nil, userAuth(f.jwt))
	if rec.Code != http.StatusOK || decodeBody(t, rec)["match"] != false {
		t.Fatalf("verify mismatch: %d %s", rec.Code, rec.Body.String())
	}
	f.db.Where("backup_run_id = ?", run.ID).First(&art)
	if art.VerificationStatus != models.ArtifactVerifyFail {
		t.Errorf("mismatch must persist FAILED, got %s", art.VerificationStatus)
	}
}
