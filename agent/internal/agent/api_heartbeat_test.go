package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Heartbeat must report the running binary version (dashboard freshness)
// with the agent credential headers, and surface auth rejection.
func TestHeartbeatSendsVersion(t *testing.T) {
	var gotBody map[string]string
	var gotAuth, gotAgentID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent/heartbeat" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		gotAgentID = r.Header.Get("X-Agent-ID")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok-123")
	c.SetAgentID("agent-123")
	if err := c.Heartbeat(); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if gotBody["agent_version"] != Version {
		t.Errorf("heartbeat must carry binary version %q, got %v", Version, gotBody)
	}
	if gotAuth != "Bearer tok-123" || gotAgentID != "agent-123" {
		t.Errorf("heartbeat must carry agent credential headers, got %q / %q", gotAuth, gotAgentID)
	}
}

func TestHeartbeatSurfacesRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"agent revoked"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "dead-token")
	if err := c.Heartbeat(); err == nil {
		t.Error("revoked credential must surface an error (runner stops on it)")
	}
}
