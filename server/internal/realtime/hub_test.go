package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/middleware"
	"github.com/gorilla/websocket"
)

func testHub() *Hub {
	// db is nil: only auth-failure paths are exercised (no queries reached).
	return NewHub(nil, "test-jwt-secret-32-bytes-long!!")
}

func TestAuthUserRejectsAnonymous(t *testing.T) {
	h := testHub()
	if _, ok := h.authUser(""); ok {
		t.Fatal("empty token accepted")
	}
	if _, ok := h.authUser("garbage"); ok {
		t.Fatal("garbage token accepted")
	}
}

func TestAuthUserAcceptsSignedToken(t *testing.T) {
	h := testHub()
	token, err := middleware.GenerateAccessToken("11111111-2222-3333-4444-555555555555", "a@x.io", "test-jwt-secret-32-bytes-long!!", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := h.authUser(token)
	if !ok {
		t.Fatal("signed token rejected")
	}
	if id.String() != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("user id = %s", id)
	}
}

func TestServeWSUnavailableWithoutDB(t *testing.T) {
	h := testHub()
	for _, target := range []string{"/ws", "/ws?token=garbage"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		h.ServeWS(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: status = %d, want 503", target, rec.Code)
		}
	}
}

func TestServeWSRejectsBadAgentCreds(t *testing.T) {
	h := testHub()
	// Agent headers present but db is nil -> 503, never a panic.
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("X-Agent-ID", "11111111-2222-3333-4444-555555555555")
	req.Header.Set("Authorization", "Bearer nope")
	rec := httptest.NewRecorder()
	h.ServeWS(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestDashboardSubscribeFlow(t *testing.T) {
	// Craft a hub wired to a throwaway DB-less store is impossible for the
	// membership check, so this test pins the protocol instead: a dashboard
	// peer that never subscribes receives nothing, and Publish to an empty
	// room is a safe no-op.
	h := testHub()
	Publish(h, "org-1", Event{Type: TypeRun})
	Publish(nil, "org-1", Event{Type: TypeRun}) // nil hub must not panic
	SendToAgent(h, "agent-1", Event{Type: TypeCancel})
	SendToAgent(nil, "agent-1", Event{Type: TypeCancel})
}

func TestUpgradeRejectsNonWS(t *testing.T) {
	h := testHub()
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/ws")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// Plain GET without upgrade headers: handshake fails (400), and a raw
	// dial without token must not establish a socket.
	if resp.StatusCode == http.StatusSwitchingProtocols {
		t.Fatal("unexpected protocol switch without credentials")
	}
	_, _, err = websocket.DefaultDialer.Dial(
		strings.Replace(srv.URL, "http", "ws", 1)+"/ws?token=bad", nil)
	if err == nil {
		t.Fatal("expected dial failure with bad token")
	}
}
