package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EnrollmentClient talks to the VaultGuard enrollment API with plain
// net/http (no agent-package dependency beyond types). The enrollment token
// is never logged and never embedded in returned errors.
type EnrollmentClient struct {
	ServerURL  string
	HTTP       *http.Client
	SkipVerify bool // lab-only self-signed escape hatch, surfaced in UI
}

type enrollResult struct {
	AgentID    string `json:"agent_id"`
	AgentToken string `json:"agent_token"`
}

// CheckConnectivity GETs the public health endpoint before enrollment.
func (c *EnrollmentClient) CheckConnectivity() error {
	if err := ValidateServerURL(c.ServerURL); err != nil {
		return err
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Get(JoinURL(c.ServerURL, "health"))
	if err != nil {
		return fmt.Errorf("cannot reach VaultGuard server: %s", sanitizeError(err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return fmt.Errorf("server answered %d, expected the VaultGuard API", resp.StatusCode)
	}
	return nil
}

// Enroll exchanges the one-time token for the permanent credential.
func (c *EnrollmentClient) Enroll(token string, info MachineInfo) (agentID, agentToken string, err error) {
	if strings.TrimSpace(token) == "" {
		return "", "", fmt.Errorf("enrollment token is required")
	}
	body, _ := json.Marshal(map[string]string{
		"enrollment_token": strings.TrimSpace(token),
		"machine_name":     info.MachineName,
		"platform":         info.Platform,
		"architecture":     info.Architecture,
		"agent_version":    info.AgentVersion,
		"os":               info.OS,
	})
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Post(JoinURL(c.ServerURL, "agents", "enroll"), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("enrollment request failed: %s", sanitizeError(err.Error()))
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", fmt.Errorf("enrollment token invalid or expired — generate a fresh one in the dashboard")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("enrollment failed (server answered %d)", resp.StatusCode)
	}
	var out enrollResult
	if err := json.Unmarshal(data, &out); err != nil {
		return "", "", fmt.Errorf("enrollment response unreadable")
	}
	if out.AgentID == "" || out.AgentToken == "" {
		return "", "", fmt.Errorf("enrollment response incomplete")
	}
	return out.AgentID, out.AgentToken, nil
}

// sanitizeError bounds error text and guarantees no token material: callers
// pass only transport errors (which never contain the token), and the token
// itself is never interpolated into any message in this package.
func sanitizeError(msg string) string {
	if i := strings.Index(msg, "\n"); i >= 0 {
		msg = msg[:i]
	}
	msg = strings.TrimSpace(msg)
	if len(msg) > 200 {
		msg = msg[:200]
	}
	if msg == "" {
		msg = "unknown error"
	}
	return msg
}
