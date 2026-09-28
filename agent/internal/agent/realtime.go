package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient holds the agent's persistent socket to the server. It serves two
// purposes: receiving instant cancel commands (no waiting for the next poll)
// and streaming lifecycle/error lines up for the dashboard's live log.
// A nil *WSClient is safe: every method no-ops, and the agent keeps working
// over plain HTTP polling.
type WSClient struct {
	serverURL  string
	agentID    string
	token      string
	skipVerify bool

	mu        sync.Mutex
	conn      *websocket.Conn
	cancelled map[string]bool
}

// NewWSClient builds (but does not connect) the socket client.
func NewWSClient(serverURL, agentID, token string, skipVerify bool) *WSClient {
	return &WSClient{
		serverURL:  serverURL,
		agentID:    agentID,
		token:      token,
		skipVerify: skipVerify,
		cancelled:  make(map[string]bool),
	}
}

// Cancelled reports whether a cancel command arrived for runID.
func (w *WSClient) Cancelled(runID string) bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.cancelled[runID]
}

func (w *WSClient) markCancelled(runID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cancelled[runID] = true
}

func (w *WSClient) forget(runID string) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.cancelled, runID)
}

// Log streams one lifecycle line for runID to dashboards watching it.
// Non-blocking: drops the line when offline.
func (w *WSClient) Log(runID, line string) {
	if w == nil || runID == "" || line == "" {
		return
	}
	w.mu.Lock()
	conn := w.conn
	w.mu.Unlock()
	if conn == nil {
		return
	}
	msg, _ := json.Marshal(map[string]interface{}{
		"type":   "log",
		"run_id": runID,
		"line":   line,
	})
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		w.mu.Lock()
		if w.conn == conn {
			w.conn.Close()
			w.conn = nil
		}
		w.mu.Unlock()
	}
}

func wsURL(serverURL string) string {
	s := strings.TrimSuffix(serverURL, "/") + "/ws"
	if strings.HasPrefix(s, "https://") {
		return "wss://" + strings.TrimPrefix(s, "https://")
	}
	return "ws://" + strings.TrimPrefix(s, "http://")
}

// Run keeps the socket connected, reconnecting with backoff until ctx ends.
func (w *WSClient) Run(ctx context.Context, onCancel func(runID string)) {
	backoff := 5 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if w.dialAndServe(ctx, onCancel) {
			return // context cancelled
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < time.Minute {
			backoff *= 2
		}
	}
}

// dialAndServe returns true only when ctx ended (don't reconnect).
func (w *WSClient) dialAndServe(ctx context.Context, onCancel func(runID string)) bool {
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	if w.skipVerify {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit lab opt-in
	}
	header := http.Header{}
	header.Set("X-Agent-ID", w.agentID)
	header.Set("Authorization", "Bearer "+w.token)
	conn, _, err := dialer.DialContext(ctx, wsURL(w.serverURL), header)
	if err != nil {
		log.Printf("agent socket: dial: %v", err)
		return false
	}
	w.mu.Lock()
	w.conn = conn
	w.mu.Unlock()
	log.Printf("agent socket: connected")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ev struct {
				Type    string                 `json:"type"`
				Payload map[string]interface{} `json:"payload"`
			}
			if err := json.Unmarshal(raw, &ev); err != nil {
				continue
			}
			if ev.Type == "cancel" {
				if id, _ := ev.Payload["run_id"].(string); id != "" {
					w.markCancelled(id)
					if onCancel != nil {
						onCancel(id)
					}
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
	w.mu.Lock()
	if w.conn == conn {
		w.conn.Close()
		w.conn = nil
	}
	w.mu.Unlock()
	conn.Close()
	return ctx.Err() != nil
}
