package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// --- one-time token hashing: deterministic, never equal to plaintext ---

func TestSha256Hex(t *testing.T) {
	h1 := sha256Hex("enrollment-token-abc")
	h2 := sha256Hex("enrollment-token-abc")
	if h1 != h2 {
		t.Error("token hash must be deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(h1))
	}
	if strings.Contains(h1, "enrollment-token-abc") {
		t.Error("hash must not contain plaintext")
	}
	if sha256Hex("token-a") == sha256Hex("token-b") {
		t.Error("different tokens must hash differently")
	}
}

// --- permanent credential minting: unique tokens, bcrypt-verifiable ---

func TestMintAgentToken(t *testing.T) {
	tok, hash, err := mintAgentToken()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if tok == "" || hash == "" {
		t.Fatal("empty token or hash")
	}
	if strings.Contains(hash, tok) {
		t.Error("hash must not embed the plaintext token")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(tok)); err != nil {
		t.Error("minted hash must verify against the minted token")
	}
	tok2, _, err := mintAgentToken()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if tok == tok2 {
		t.Error("minted tokens must be unique")
	}
}

// --- enrollment expiry predicate ---

func TestEnrollmentExpired(t *testing.T) {
	now := time.Now()
	if enrollmentExpired(now.Add(time.Minute), now) {
		t.Error("future expiry must not be expired")
	}
	if !enrollmentExpired(now.Add(-time.Minute), now) {
		t.Error("past expiry must be expired")
	}
	if !enrollmentExpired(now, now) {
		t.Error("exact-now expiry must be expired (strict Before)")
	}
}

// --- lifecycle matrix ---

func agentWithState(revoked bool, tokenHash *string, status models.AgentStatus, lastSeen *time.Time) models.Agent {
	a := models.Agent{Status: status, LastSeenAt: lastSeen}
	if revoked {
		t := time.Now()
		a.RevokedAt = &t
	}
	a.TokenHash = tokenHash
	return a
}

func TestAgentLifecycle(t *testing.T) {
	hash := "bcrypt-hash"
	recent := time.Now().Add(-time.Minute)
	stale := time.Now().Add(-time.Hour)

	cases := []struct {
		name string
		in   models.Agent
		want string
	}{
		{"pending never enrolled", agentWithState(false, nil, models.AgentOffline, nil), models.LifecyclePending},
		{"revoked beats everything", agentWithState(true, &hash, models.AgentOnline, &recent), models.LifecycleRevoked},
		{"revoked pending", agentWithState(true, nil, models.AgentOffline, nil), models.LifecycleRevoked},
		{"recent heartbeat online", agentWithState(false, &hash, models.AgentOffline, &recent), models.LifecycleOnline},
		{"stale heartbeat offline", agentWithState(false, &hash, models.AgentOffline, &stale), models.LifecycleOffline},
		{"no heartbeat offline", agentWithState(false, &hash, models.AgentOffline, nil), models.LifecycleOffline},
	}
	for _, tt := range cases {
		if got := tt.in.Lifecycle(); got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}

// --- lifecycle DTO: additive, no secrets ---

func TestAgentWithLifecycleNoSecrets(t *testing.T) {
	hash := "bcrypt-hash"
	key := "digest"
	a := models.Agent{Status: models.AgentOffline, TokenHash: &hash, RegistrationKey: &key}
	dto := agentWithLifecycle(a)
	for k := range dto {
		if k == "token_hash" || k == "registration_key" || k == "agent_token" {
			t.Errorf("lifecycle DTO must not contain %q", k)
		}
	}
	if dto["lifecycle"] != models.LifecycleOffline {
		t.Errorf("wrong lifecycle: %v", dto["lifecycle"])
	}
}
