package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"
)

// JobConfig mirrors the server's agentJobConfigDTO.
type JobConfig struct {
	JobID           string `json:"job_id"`
	Name            string `json:"name"`
	SourceType      string `json:"source_type"`
	SourcePath      string `json:"source_path"`
	SourceDatabase  string `json:"source_database"`
	IncludePatterns string `json:"include_patterns"`
	ExcludePatterns string `json:"exclude_patterns"`
	Mode            string `json:"mode"`
	Encrypted       bool   `json:"encrypted"`
	RetentionDays   int    `json:"retention_days"`
	StorageType     string `json:"storage_type"`
	StorageBucket   string `json:"storage_bucket"`
	StorageRegion   string `json:"storage_region"`
	StorageEndpoint string `json:"storage_endpoint"`
	StoragePath     string `json:"storage_path"`
}

type Run struct {
	ID          string `json:"id"`
	BackupJobID string `json:"backup_job_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type ClaimRunResponse struct {
	Run    Run       `json:"run"`
	Config JobConfig `json:"config"`
}

type Restore struct {
	ID              string `json:"id"`
	BackupRunID     string `json:"backup_run_id"`
	DestinationPath string `json:"destination_path"`
	TargetDatabase  string `json:"target_database"`
	Status          string `json:"status"`
	StoragePath     string `json:"storage_path"`
	StorageType     string `json:"storage_type"`
	// DataKey is the raw base64 data key for encrypted backups.
	DataKey   string `json:"data_key"`
	// SourceType/SourceDatabase identify what is being restored.
	SourceType     string `json:"source_type"`
	SourceDatabase string `json:"source_database"`
	CreatedAt string `json:"created_at"`
}

type Client struct {
	server  string
	agentID string
	token   string
	http    *http.Client
}

func NewClient(server, token string) *Client {
	return &Client{
		server: server,
		token:  token,
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

// SetAgentID attaches the agent identity sent as X-Agent-ID on every
// authenticated request (required by the server alongside the token).
func (c *Client) SetAgentID(id string) {
	c.agentID = id
}

func (c *Client) do(method, path string, body interface{}, authed bool) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rdr = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.server+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authed {
		req.Header.Set("Authorization", "Bearer "+c.token)
		if c.agentID != "" {
			req.Header.Set("X-Agent-ID", c.agentID)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error != "" {
			return nil, resp.StatusCode, fmt.Errorf("server %d: %s", resp.StatusCode, apiErr.Error)
		}
		return nil, resp.StatusCode, fmt.Errorf("server %d: %s", resp.StatusCode, string(data))
	}
	return data, resp.StatusCode, nil
}

type registerResponse struct {
	AgentID    string `json:"agent_id"`
	AgentToken string `json:"agent_token"`
}

// Register performs the one-time self-registration with the registration key.
func (c *Client) Register(regKey, hostname, version string) (*registerResponse, error) {
	data, _, err := c.do("POST", "/api/agents/register", map[string]string{
		"registration_key": regKey,
		"hostname":         hostname,
		"os":               runtime.GOOS,
		"version":          version,
		"ip_address":       outboundIP(),
	}, false)
	if err != nil {
		return nil, err
	}
	var out registerResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PendingRuns() ([]Run, error) {
	data, _, err := c.do("GET", "/api/agent/runs", nil, true)
	if err != nil {
		return nil, err
	}
	var out []Run
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Run{}
	}
	return out, nil
}

func (c *Client) ClaimRun(id string) (*ClaimRunResponse, error) {
	data, code, err := c.do("POST", "/api/agent/runs/"+id+"/claim", nil, true)
	if err != nil {
		if code == 409 {
			return nil, errClaimed
		}
		return nil, err
	}
	var out ClaimRunResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

var errClaimed = fmt.Errorf("already claimed")

type RunStatusUpdate struct {
	Status          string `json:"status"`
	BytesRead       int64  `json:"bytes_read,omitempty"`
	BytesCompressed int64  `json:"bytes_compressed,omitempty"`
	BytesUploaded   int64  `json:"bytes_uploaded,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
	StoragePath     string `json:"storage_path,omitempty"`
	Checksum        string `json:"checksum,omitempty"`
	// DataKey is the base64 per-backup AES-256 data key, sent once with the
	// completion report for encrypted backups.
	DataKey string `json:"data_key,omitempty"`
}

func (c *Client) UpdateRunStatus(runID string, u RunStatusUpdate) error {
	_, _, err := c.do("PUT", "/api/backup-runs/"+runID+"/status", u, true)
	return err
}

type ArtifactResponse struct {
	ID string `json:"id"`
}

func (c *Client) RegisterArtifact(runID, name string, size int64, checksum, storagePath string) (*ArtifactResponse, error) {
	data, _, err := c.do("POST", "/api/backup-runs/"+runID+"/artifacts", map[string]interface{}{
		"name":         name,
		"size":         size,
		"checksum":     checksum,
		"storage_path": storagePath,
	}, true)
	if err != nil {
		return nil, err
	}
	var out ArtifactResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PendingRestores() ([]Restore, error) {
	data, _, err := c.do("GET", "/api/agent/restores", nil, true)
	if err != nil {
		return nil, err
	}
	var out []Restore
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Restore{}
	}
	return out, nil
}

func (c *Client) ClaimRestore(id string) (*Restore, error) {
	data, code, err := c.do("POST", "/api/agent/restores/"+id+"/claim", nil, true)
	if err != nil {
		if code == 409 {
			return nil, errClaimed
		}
		return nil, err
	}
	var out Restore
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateRestoreStatus(id, status, errMsg string) error {
	_, _, err := c.do("PUT", "/api/restores/"+id+"/status", map[string]string{
		"status":        status,
		"error_message": errMsg,
	}, true)
	return err
}

// outboundIP returns the preferred outbound IP without sending traffic.
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}
