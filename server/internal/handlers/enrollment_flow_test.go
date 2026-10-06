// Package handlers_test exercises the enrollment lifecycle over HTTP against
// a real database: token generation, enrollment, single-use, expiry,
// revocation, heartbeat, rename, org isolation, and response hygiene.
// DB-gated via testutil.OpenDB (skips without DATABASE_URL; Jenkins runs
// these against ephemeral Postgres).
package handlers_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/api"
	"github.com/backup-saas/server/internal/config"
	"github.com/backup-saas/server/internal/grpcserver"
	"github.com/backup-saas/server/internal/middleware"
	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const testJWTSecret = "enrollment-flow-test-secret-must-be-long-enough"

type enrollFixture struct {
	db     *gorm.DB
	router *gin.Engine
	org    models.Organization
	owner  models.User
	jwt    string
}

func setupEnrollFixture(t *testing.T, orgName, email string) *enrollFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.OpenDB(t)
	cfg := &config.Config{
		JWTSecret:             testJWTSecret,
		JWTRefreshSecret:      testJWTSecret + "-refresh",
		AccessTokenTTLMinutes: 60,
		RefreshTokenTTLDays:   7,
		EncryptionKeyBytes:    bytes.Repeat([]byte{7}, 32),
		CORSOrigin:            "*",
	}
	grpcSrv := grpcserver.NewServer(db, cfg.EncryptionKeyBytes, nil)
	hub := realtime.NewHub(db, cfg.JWTSecret, "*")
	r := api.NewRouter(db, cfg, grpcSrv, hub)

	user := testutil.CreateUser(t, db, email, "password123")
	org := testutil.CreateOrg(t, db, orgName, user.ID)
	jwt, err := middleware.GenerateAccessToken(user.ID.String(), user.Email, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("mint jwt: %v", err)
	}
	return &enrollFixture{db: db, router: r, org: org, owner: user, jwt: jwt}
}

func doReq(t *testing.T, r *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %d body %q: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func userAuth(jwt string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + jwt}
}

func agentAuth(agentID, token string) map[string]string {
	return map[string]string{"X-Agent-ID": agentID, "Authorization": "Bearer " + token}
}

// mintToken provisions a token via the API and returns plaintext + agent ID.
func mintToken(t *testing.T, f *enrollFixture) (raw, agentID string) {
	t.Helper()
	rec := doReq(t, f.router, "POST",
		"/vaultguard/api/organizations/"+f.org.ID.String()+"/agents/enrollment-token",
		map[string]interface{}{"name": "test-pc", "expires_minutes": 15}, userAuth(f.jwt))
	if rec.Code != http.StatusCreated {
		t.Fatalf("mint token: %d %s", rec.Code, rec.Body.String())
	}
	out := decodeBody(t, rec)
	raw, _ = out["enrollment_token"].(string)
	agentID, _ = out["agent_id"].(string)
	if raw == "" || agentID == "" {
		t.Fatalf("mint response missing fields: %s", rec.Body.String())
	}
	t.Cleanup(func() {
		var aid uuid.UUID
		if id, err := uuid.Parse(agentID); err == nil {
			aid = id
			f.db.Unscoped().Where("agent_id = ?", aid).Delete(&models.Machine{})
			f.db.Unscoped().Where("agent_id = ?", aid).Delete(&models.EnrollmentToken{})
			f.db.Unscoped().Where("id = ?", aid).Delete(&models.Agent{})
		}
	})
	return raw, agentID
}

func enrollBody(token string) map[string]interface{} {
	return map[string]interface{}{
		"enrollment_token": token,
		"machine_name":     "TEST-PC",
		"platform":         "windows",
		"architecture":     "amd64",
		"agent_version":    "9.9.9-test",
		"hostname":         "TEST-PC",
		"os":               "windows",
		"ip_address":       "10.0.0.9",
	}
}

func sha256HexLocal(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// --- 1. valid enrollment: success shape, DB state, one-time secret hygiene ---

func TestEnrollFlow_Success(t *testing.T) {
	f := setupEnrollFixture(t, "Enroll Org", "enroll-ok@test.com")
	raw, agentID := mintToken(t, f)

	rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("enroll: %d %s", rec.Code, rec.Body.String())
	}
	out := decodeBody(t, rec)
	agentToken, _ := out["agent_token"].(string)
	if agentToken == "" || out["agent_id"] != agentID {
		t.Fatalf("enroll response wrong: %s", rec.Body.String())
	}
	// Response hygiene: permanent credential shown once (by design), but the
	// enrollment token and any hash must never appear.
	if bytes.Contains(rec.Body.Bytes(), []byte(raw)) {
		t.Error("enroll response leaks the enrollment token")
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("token_hash")) {
		t.Error("enroll response leaks token_hash")
	}

	// DB state: token consumed, agent credentialed (hash-only), identity set.
	var tok models.EnrollmentToken
	if err := f.db.Where("token_hash = ?", sha256HexLocal(raw)).First(&tok).Error; err != nil {
		t.Fatalf("token row: %v", err)
	}
	if tok.UsedAt == nil {
		t.Error("token must be marked used")
	}
	var agent models.Agent
	if err := f.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		t.Fatalf("agent row: %v", err)
	}
	if agent.TokenHash == nil || *agent.TokenHash == agentToken {
		t.Fatal("server must store only the bcrypt hash, never the credential")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*agent.TokenHash), []byte(agentToken)); err != nil {
		t.Error("stored hash must verify the issued credential")
	}
	if agent.Name != "TEST-PC" || agent.Version != "9.9.9-test" || agent.Platform != "windows" || agent.InstalledAt == nil {
		t.Errorf("agent identity not recorded: %+v", agent)
	}
}

// --- 2/4/5. reuse, invalid, expired share one generic 401 (no oracle) ---

func invalidEnrollMessage(t *testing.T) string {
	t.Helper()
	f := setupEnrollFixture(t, "Oracle Org", "oracle@test.com")
	rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll",
		enrollBody("definitely-not-a-real-token"), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token: %d", rec.Code)
	}
	return decodeBody(t, rec)["error"].(string)
}

func TestEnrollFlow_ReusedInvalidExpiredShareMessage(t *testing.T) {
	f := setupEnrollFixture(t, "Reuse Org", "reuse@test.com")
	want := invalidEnrollMessage(t)

	// Reused: enroll once (consumes), then again.
	raw, _ := mintToken(t, f)
	rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("first enroll: %d", rec.Code)
	}
	rec = doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused token: %d, want 401", rec.Code)
	}
	if got := decodeBody(t, rec)["error"].(string); got != want {
		t.Errorf("reused message %q differs from invalid %q (oracle)", got, want)
	}

	// Expired: insert a spent-clock token directly (hash-only, as production stores).
	expiredRaw := "expired-token-" + uuid.New().String()
	agentID := uuid.New()
	f.db.Create(&models.Agent{
		Base: models.Base{ID: agentID}, OrganizationID: f.org.ID, Name: "expiry-pc",
		Status: models.AgentOffline,
	})
	t.Cleanup(func() {
		f.db.Unscoped().Where("agent_id = ?", agentID).Delete(&models.EnrollmentToken{})
		f.db.Unscoped().Where("id = ?", agentID).Delete(&models.Agent{})
	})
	f.db.Create(&models.EnrollmentToken{
		Base: models.Base{ID: uuid.New()}, OrganizationID: f.org.ID, AgentID: &agentID,
		TokenHash: sha256HexLocal(expiredRaw), ExpiresAt: time.Now().Add(-time.Minute),
	})
	rec = doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(expiredRaw), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired token: %d, want 401", rec.Code)
	}
	if got := decodeBody(t, rec)["error"].(string); got != want {
		t.Errorf("expired message %q differs from invalid %q (oracle)", got, want)
	}
}

// --- 6. concurrent enrollment: exactly one winner ---

func TestEnrollFlow_ConcurrentSingleUse(t *testing.T) {
	f := setupEnrollFixture(t, "Race Org", "race@test.com")
	raw, agentID := mintToken(t, f)

	const n = 8
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw), nil)
			codes[idx] = rec.Code
		}(i)
	}
	wg.Wait()
	wins := 0
	for _, c := range codes {
		if c == http.StatusOK {
			wins++
		} else if c != http.StatusUnauthorized {
			t.Errorf("unexpected code %d (want 200 once, 401 rest)", c)
		}
	}
	if wins != 1 {
		t.Errorf("expected exactly 1 winner, got %d of %d: %v", wins, n, codes)
	}
	var tok models.EnrollmentToken
	f.db.Where("token_hash = ?", sha256HexLocal(raw)).First(&tok)
	if tok.UsedAt == nil {
		t.Error("winning enrollment must mark the token used")
	}
	_ = agentID
}

// --- heartbeat, version, restart retention, revocation ---

func TestEnrollFlow_HeartbeatAndRevoke(t *testing.T) {
	f := setupEnrollFixture(t, "Beat Org", "beat@test.com")
	raw, agentID := mintToken(t, f)
	rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("enroll: %d", rec.Code)
	}
	agentToken := decodeBody(t, rec)["agent_token"].(string)
	ah := agentAuth(agentID, agentToken)

	// Heartbeat reports version; restart (fresh request, same credential) works.
	for i := 0; i < 2; i++ {
		rec = doReq(t, f.router, "POST", "/vaultguard/api/agent/heartbeat",
			map[string]interface{}{"agent_version": "9.9.9-test"}, ah)
		if rec.Code != http.StatusOK {
			t.Fatalf("heartbeat %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	var agent models.Agent
	f.db.Where("id = ?", agentID).First(&agent)
	if agent.LastSeenAt == nil || time.Since(*agent.LastSeenAt) > time.Minute {
		t.Error("heartbeat must refresh last_seen_at")
	}
	if agent.Version != "9.9.9-test" {
		t.Errorf("heartbeat must update version, got %q", agent.Version)
	}
	if agent.Lifecycle() != models.LifecycleOnline {
		t.Errorf("recent heartbeat must read ONLINE, got %s", agent.Lifecycle())
	}

	// Legacy heartbeat without body still works.
	rec = doReq(t, f.router, "POST", "/vaultguard/api/agent/heartbeat", nil, ah)
	if rec.Code != http.StatusOK {
		t.Errorf("bodiless heartbeat: %d", rec.Code)
	}

	// Revoke: heartbeat, job claims, and restore claims all reject.
	rec = doReq(t, f.router, "POST",
		"/vaultguard/api/organizations/"+f.org.ID.String()+"/agents/"+agentID+"/revoke",
		map[string]interface{}{}, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body.String())
	}
	f.db.Where("id = ?", agentID).First(&agent)
	if agent.RevokedAt == nil || agent.Lifecycle() != models.LifecycleRevoked {
		t.Error("revoked agent must stay REVOKED regardless of recency")
	}
	for path, method := range map[string]string{
		"/vaultguard/api/agent/heartbeat":               "POST",
		"/vaultguard/api/agent/runs":                    "GET",
		"/vaultguard/api/agent/restores":                "GET",
		"/vaultguard/api/backup-runs/00000000-0000-0000-0000-000000000000/status": "PUT",
	} {
		rec = doReq(t, f.router, method, path, map[string]interface{}{"status": "RUNNING"}, ah)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s after revoke: %d, want 401", method, path, rec.Code)
		}
	}
}

// --- rename ---

func TestEnrollFlow_Rename(t *testing.T) {
	f := setupEnrollFixture(t, "Rename Org", "rename@test.com")
	_, agentID := mintToken(t, f)
	path := "/vaultguard/api/organizations/" + f.org.ID.String() + "/agents/" + agentID

	rec := doReq(t, f.router, "PUT", path, map[string]interface{}{"name": "  Front Desk  "}, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body.String())
	}
	var agent models.Agent
	f.db.Where("id = ?", agentID).First(&agent)
	if agent.Name != "Front Desk" {
		t.Errorf("name not trimmed/updated: %q", agent.Name)
	}
	for _, bad := range []map[string]interface{}{{"name": ""}, {"name": "   "}, {"name": strings.Repeat("x", 101)}} {
		rec = doReq(t, f.router, "PUT", path, bad, userAuth(f.jwt))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("bad name: %d, want 400", rec.Code)
		}
	}
}

// --- token revoke + list hygiene ---

func TestEnrollFlow_TokenRevoke(t *testing.T) {
	f := setupEnrollFixture(t, "TokenRevoke Org", "tokrev@test.com")
	raw, agentID := mintToken(t, f)

	// List shows metadata, never hashes.
	rec := doReq(t, f.router, "GET",
		"/vaultguard/api/organizations/"+f.org.ID.String()+"/agents/enrollment-tokens", nil, userAuth(f.jwt))
	if rec.Code != http.StatusOK {
		t.Fatalf("list tokens: %d", rec.Code)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("token_hash")) {
		t.Error("token list leaks hashes")
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) == 0 {
		t.Fatalf("token list unreadable: %s", rec.Body.String())
	}
	tokID, _ := list[0]["id"].(string)
	if tokID == "" {
		t.Fatalf("token list missing id: %s", rec.Body.String())
	}

	tokPath := "/vaultguard/api/organizations/" + f.org.ID.String() + "/agents/enrollment-tokens/" + tokID
	rec = doReq(t, f.router, "DELETE", tokPath, nil, userAuth(f.jwt))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke token: %d %s", rec.Code, rec.Body.String())
	}
	// Revoked token cannot enroll; repeat revoke is uniform 404.
	rec = doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("revoked token enroll: %d, want 401", rec.Code)
	}
	revokedMsg := decodeBody(t, rec)["error"]
	rec = doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll",
		enrollBody("no-such-token"), nil)
	if rec.Code != http.StatusUnauthorized || decodeBody(t, rec)["error"] != revokedMsg {
		t.Error("revoked vs invalid must share one generic 401 message (no oracle)")
	}
	// Auditability: the revoked row is kept with revoked_at, not deleted.
	var kept models.EnrollmentToken
	if err := f.db.Where("token_hash = ?", sha256HexLocal(raw)).First(&kept).Error; err != nil {
		t.Errorf("revoked token row must be retained: %v", err)
	} else if kept.RevokedAt == nil {
		t.Error("revoked token row must carry revoked_at")
	}
	rec = doReq(t, f.router, "DELETE", tokPath, nil, userAuth(f.jwt))
	if rec.Code != http.StatusNotFound {
		t.Errorf("repeat revoke: %d, want 404", rec.Code)
	}

	// Used tokens cannot be revoked either (uniform 404, no oracle).
	raw2, _ := mintToken(t, f)
	if rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(raw2), nil); rec.Code != http.StatusOK {
		t.Fatalf("second enroll: %d", rec.Code)
	}
	rec = doReq(t, f.router, "GET",
		"/vaultguard/api/organizations/"+f.org.ID.String()+"/agents/enrollment-tokens", nil, userAuth(f.jwt))
	var list2 []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list2)
	for _, item := range list2 {
		if item["used_at"] != nil {
			tid, _ := item["id"].(string)
			rec = doReq(t, f.router, "DELETE",
				"/vaultguard/api/organizations/"+f.org.ID.String()+"/agents/enrollment-tokens/"+tid, nil, userAuth(f.jwt))
			if rec.Code != http.StatusNotFound {
				t.Errorf("used token revoke: %d, want 404", rec.Code)
			}
		}
	}
	_ = agentID
}

// --- org isolation across every new endpoint ---

func TestEnrollFlow_OrgIsolation(t *testing.T) {
	fa := setupEnrollFixture(t, "Iso A", "iso-a@test.com")
	fb := setupEnrollFixture(t, "Iso B", "iso-b@test.com")
	rawA, agentA := mintToken(t, fa)

	// Org B admin sees nothing of org A: agent get/rename/revoke, token list/revoke.
	cases := []struct {
		method, path string
		body         interface{}
	}{
		{"GET", "/vaultguard/api/organizations/" + fb.org.ID.String() + "/agents/" + agentA, nil},
		{"PUT", "/vaultguard/api/organizations/" + fb.org.ID.String() + "/agents/" + agentA, map[string]interface{}{"name": "x"}},
		{"POST", "/vaultguard/api/organizations/" + fb.org.ID.String() + "/agents/" + agentA + "/revoke", map[string]interface{}{}},
	}
	for _, c := range cases {
		rec := doReq(t, fa.router, c.method, c.path, c.body, userAuth(fb.jwt))
		if rec.Code == http.StatusOK || rec.Code == http.StatusCreated {
			t.Errorf("cross-org %s %s succeeded: %d", c.method, c.path, rec.Code)
		}
	}
	// Org B token list must not contain org A's token (by agent link).
	rec := doReq(t, fa.router, "GET",
		"/vaultguard/api/organizations/"+fb.org.ID.String()+"/agents/enrollment-tokens", nil, userAuth(fb.jwt))
	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, item := range list {
		if item["agent_id"] == agentA {
			t.Error("org B token list exposes org A agent link")
		}
	}
	// Org A's own token still enrolls fine (cross-check the flow wasn't broken).
	if rec := doReq(t, fa.router, "POST", "/vaultguard/api/agents/enroll", enrollBody(rawA), nil); rec.Code != http.StatusOK {
		t.Errorf("org A enroll: %d", rec.Code)
	}
}

// --- enrollment rate limiting (brute-force/enumeration backstop) ---

func TestEnrollFlow_RateLimited(t *testing.T) {
	f := setupEnrollFixture(t, "Rate Org", "ratelimit@test.com")
	limited := false
	for i := 0; i < 25; i++ {
		rec := doReq(t, f.router, "POST", "/vaultguard/api/agents/enroll",
			enrollBody("nope-not-a-token"), nil)
		if rec.Code == http.StatusTooManyRequests {
			limited = true
			break
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d, want 401-or-429", i, rec.Code)
		}
	}
	if !limited {
		t.Error("expected 429s after exceeding the enroll budget")
	}
}

// --- offline detection is recency-derived (pure lifecycle matrix) ---

func TestEnrollFlow_OfflineLifecycle(t *testing.T) {
	fresh := time.Now()
	stale := time.Now().Add(-time.Hour)
	mk := func(lastSeen *time.Time, status models.AgentStatus, revoked bool) *models.Agent {
		a := &models.Agent{Status: status, LastSeenAt: lastSeen}
		h := "hash"
		a.TokenHash = &h
		if revoked {
			t := time.Now()
			a.RevokedAt = &t
		}
		return a
	}
	if got := mk(&fresh, models.AgentOffline, false).Lifecycle(); got != models.LifecycleOnline {
		t.Errorf("fresh heartbeat must read ONLINE, got %s", got)
	}
	if got := mk(&stale, models.AgentOnline, false).Lifecycle(); got != models.LifecycleOffline {
		t.Errorf("stale heartbeat must read OFFLINE, got %s", got)
	}
	if got := mk(&fresh, models.AgentOnline, true).Lifecycle(); got != models.LifecycleRevoked {
		t.Errorf("revoked must win over recency, got %s", got)
	}
	pending := mk(nil, models.AgentOffline, false)
	pending.TokenHash = nil
	if l := pending.Lifecycle(); l != models.LifecyclePending {
		t.Errorf("never-enrolled must read PENDING, got %s", l)
	}
}
