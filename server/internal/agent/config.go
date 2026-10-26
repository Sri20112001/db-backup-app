package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version is reported to the server on registration.
const Version = "0.1.0"

type Config struct {
	// Server is the API base URL, e.g. http://localhost:7541
	Server string
	// RegistrationKey is the one-time key from GenerateRegistrationToken.
	// Only needed on first run; afterwards state file holds the credentials.
	RegistrationKey string
	// StateFile persists agent_id + agent_token between restarts.
	StateFile string
	Hostname  string
	// PollInterval between server polls. Also acts as the heartbeat that
	// keeps the agent ONLINE (server marks agents stale after 3 min).
	PollInterval time.Duration
	// BrowseAddr is the loopback address of the folder browser used by
	// the dashboard picker. Empty disables it.
	BrowseAddr string
}

type state struct {
	AgentID    string `json:"agent_id"`
	AgentToken string `json:"agent_token"`
}

func defaultHostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown-host"
}

func LoadConfig() Config {
	poll := 30 * time.Second
	if v := os.Getenv("AGENT_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 5*time.Second {
			poll = d
		}
	}
	stateFile := os.Getenv("AGENT_STATE_FILE")
	if stateFile == "" {
		exe, err := os.Executable()
		if err != nil {
			stateFile = "agent-state.json"
		} else {
			stateFile = filepath.Join(filepath.Dir(exe), "agent-state.json")
		}
	}
	hostname := os.Getenv("AGENT_HOSTNAME")
	if hostname == "" {
		hostname = defaultHostname()
	}
	server := os.Getenv("AGENT_SERVER")
	if server == "" {
		server = "http://localhost:7541"
	}
	browseAddr := os.Getenv("AGENT_BROWSE_ADDR")
	if browseAddr == "" {
		if v := os.Getenv("AGENT_BROWSE_PORT"); v != "" {
			browseAddr = "127.0.0.1:" + v
		} else {
			browseAddr = "127.0.0.1:7546"
		}
	}
	// AGENT_BROWSE_ADDR=off disables the folder browser.
	if browseAddr == "off" {
		browseAddr = ""
	}
	return Config{
		Server:          server,
		RegistrationKey: os.Getenv("AGENT_REGISTRATION_KEY"),
		StateFile:       stateFile,
		Hostname:        hostname,
		PollInterval:    poll,
		BrowseAddr:      browseAddr,
	}
}

func loadState(path string) (*state, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.AgentID == "" || s.AgentToken == "" {
		return nil, fmt.Errorf("incomplete state file")
	}
	return &s, nil
}

func saveState(path string, s *state) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
