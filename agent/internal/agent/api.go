package agent

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// JobConfig mirrors the server's agentJobConfigDTO.
type JobConfig struct {
	JobID           string `json:"job_id"`
	OrganizationID  string `json:"organization_id"`
	Name            string `json:"name"`
	SourceType      string `json:"source_type"`
	ConnectionID    string `json:"connection_id,omitempty"`
	SourcePath      string `json:"source_path"`
	SourceDatabase  string `json:"source_database"`
	IncludePatterns string `json:"include_patterns"`
	ExcludePatterns string `json:"exclude_patterns"`
	Mode            string `json:"mode"`
	Encrypted       bool   `json:"encrypted"`
	RetentionDays   int    `json:"retention_days"`
	// Connection carries per-claim decrypted DB credentials for jobs that
	// reference a saved connection. Cleared from memory after use; never
	// logged. Nil for filesystem / legacy jobs (env fallback applies).
	Connection *ConnectionConfig `json:"connection,omitempty"`
	StorageType     string `json:"storage_type"`
	StorageBucket   string `json:"storage_bucket"`
	StorageRegion   string `json:"storage_region"`
	StorageEndpoint string `json:"storage_endpoint"`
	StoragePath     string `json:"storage_path"`
	// Storage is the self-contained provider config (S3 credentials
	// included, decrypted server-side per claim). New code reads this;
	// the flat Storage* fields above stay for older servers.
	Storage StorageConfig `json:"storage"`
	// ExportFormat selects the MongoDB payload (ARCHIVE/JSON/CSV).
	ExportFormat string `json:"export_format"`
}

// ConnectionConfig mirrors the server's agentConnectionDTO: per-claim
// decrypted credentials for one saved connection.
type ConnectionConfig struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// StorageConfig mirrors the server's agentStorageConfigDTO.
type StorageConfig struct {
	Type         string `json:"type"`
	Path         string `json:"path"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	Endpoint     string `json:"endpoint"`
	UsePathStyle bool   `json:"use_path_style"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
}

type Run struct {
	ID          string `json:"id"`
	BackupJobID string `json:"backup_job_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type ClaimRunResponse struct {
	Run             Run       `json:"run"`
	Config          JobConfig `json:"config"`
	CancelRequested bool      `json:"cancel_requested"`
}

type Restore struct {
	ID              string `json:"id"`
	BackupRunID     string `json:"backup_run_id"`
	DestinationPath string `json:"destination_path"`
	TargetDatabase  string `json:"target_database"`
	Status          string `json:"status"`
	StoragePath     string `json:"storage_path"`
	StorageType     string `json:"storage_type"`
	// Storage carries the provider config (S3 credentials included) so the
	// agent can download without extra calls.
	Storage StorageConfig `json:"storage"`
	// DataKey is the raw base64 data key for encrypted backups.
	DataKey   string `json:"data_key"`
	// Connection carries per-claim decrypted DB credentials for database
	// restores (copied from the source backup job). Nil for filesystem /
	// legacy restores (env fallback applies). Cleared after use, never logged.
	Connection *ConnectionConfig `json:"connection,omitempty"`
	// SourceType/SourceDatabase identify what is being restored.
	SourceType     string `json:"source_type"`
	SourceDatabase string `json:"source_database"`
	// ExportFormat selects the MongoDB restore path (ARCHIVE/JSON/CSV).
	ExportFormat string `json:"export_format"`
	CreatedAt string `json:"created_at"`
}

type Client struct {
	server  string
	agentID string
	token   string
	http    *http.Client
}

func NewClient(server, token string) *Client {
	return NewClientWithTLS(server, token, false)
}

// NewClientWithTLS optionally skips certificate verification (self-signed
// or lab CAs only — never in production).
func NewClientWithTLS(server, token string, skipVerify bool) *Client {
	transport := http.DefaultTransport
	if skipVerify {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // explicit lab opt-in
		}
	}
	return &Client{
		server: server,
		token:  token,
		http:   &http.Client{Timeout: 60 * time.Second, Transport: transport},
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

// EnrollRequest is the setup/agent enrollment payload. The enrollment token
// is single-use and short-lived; it is never logged and never appears in
// error strings.
type EnrollRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	MachineName     string `json:"machine_name"`
	Platform        string `json:"platform"`
	Architecture    string `json:"architecture"`
	AgentVersion    string `json:"agent_version"`
	Hostname        string `json:"hostname,omitempty"`
	OS              string `json:"os,omitempty"`
	IPAddress       string `json:"ip_address,omitempty"`
}

type enrollResponse struct {
	AgentID                  string `json:"agent_id"`
	AgentToken               string `json:"agent_token"`
	PollIntervalSeconds      int    `json:"poll_interval_seconds"`
	HeartbeatIntervalSeconds int    `json:"heartbeat_interval_seconds"`
}

// Enroll exchanges a one-time enrollment token for a permanent credential.
// Errors are sanitized: the token value is never included.
func (c *Client) Enroll(req EnrollRequest) (*enrollResponse, error) {
	data, _, err := c.do("POST", "/agents/enroll", req, false)
	if err != nil {
		return nil, sanitizeClientError(err)
	}
	var out enrollResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out.AgentID == "" || out.AgentToken == "" {
		return nil, fmt.Errorf("enrollment response incomplete")
	}
	return &out, nil
}

// sanitizeClientError strips any credential material from request errors.
// Server error bodies never contain the token, but transport errors echo
// URLs — keep only the status/message, never the payload.
func sanitizeClientError(err error) error {
	msg := err.Error()
	if i := strings.Index(msg, "\n"); i >= 0 {
		msg = msg[:i]
	}
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return errors.New(msg)
}

// Heartbeat posts the explicit liveness signal. The server derives
// ONLINE/OFFLINE from recency; a 401 here means revoked/invalid credential.
func (c *Client) Heartbeat() error {
	_, _, err := c.do("POST", "/agent/heartbeat", nil, true)
	return err
}

// Register performs the one-time self-registration with the registration key.
func (c *Client) Register(regKey, hostname, version string) (*registerResponse, error) {
	data, _, err := c.do("POST", "/agents/register", map[string]string{
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
	data, _, err := c.do("GET", "/agent/runs", nil, true)
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
	data, code, err := c.do("POST", "/agent/runs/"+id+"/claim", nil, true)
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

func (c *Client) UpdateRunStatus(runID string, u RunStatusUpdate) (bool, error) {
	data, _, err := c.do("PUT", "/backup-runs/"+runID+"/status", u, true)
	if err != nil {
		return false, err
	}
	var out struct {
		CancelRequested bool `json:"cancel_requested"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return false, nil // old server: no flag, assume keep going
	}
	return out.CancelRequested, nil
}

type ArtifactResponse struct {
	ID string `json:"id"`
}

func (c *Client) RegisterArtifact(runID, name string, size int64, checksum, storagePath string) (*ArtifactResponse, error) {
	data, _, err := c.do("POST", "/backup-runs/"+runID+"/artifacts", map[string]interface{}{
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
	data, _, err := c.do("GET", "/agent/restores", nil, true)
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
	data, code, err := c.do("POST", "/agent/restores/"+id+"/claim", nil, true)
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
	_, _, err := c.do("PUT", "/restores/"+id+"/status", map[string]string{
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
