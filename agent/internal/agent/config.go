package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Version is reported to the server on registration.
const Version = "0.4.0"

// PgConfig holds PostgreSQL connection parameters for pg_dump/pg_restore.
// Credentials live only in the agent environment — they are never sent to
// the server or stored in the database. Per-job config carries just the
// database name.
type PgConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// MssqlConfig holds SQL Server connection parameters for BACKUP/RESTORE
// DATABASE. Same rule as PgConfig: credentials stay on the machine, the
// job carries only the database name. Empty User selects Windows auth (-E),
// so a service account with sysadmin (or db_backupoperator) normally needs
// no extra configuration.
type MssqlConfig struct {
	// Server is "host" or "host\\INSTANCE" (default localhost).
	Server   string
	User     string
	Password string
	// BackupDir is where .bak files are written. It must be a path local to
	// the SQL Server engine and writable by its service account. Empty
	// defaults to the OS temp dir.
	BackupDir string
}

// MongoConfig holds the MongoDB connection string for mongodump/mongorestore.
// It lives only in the agent environment — never sent to the server or
// stored. Per-job config carries just the database name.
type MongoConfig struct {
	// URI is a full MongoDB connection string, e.g.
	// mongodb://user:pass@localhost:27017/?authSource=admin
	URI string
}

type Config struct {
	// Server is the API base URL, e.g. http://localhost:7541
	Server string
	// RegistrationKey is the one-time key from GenerateRegistrationToken.
	// Only needed on first run; afterwards state file holds the credentials.
	RegistrationKey string
	// EnrollmentToken is the one-time token from the enrollment API
	// (Agents → Add Agent). Preferred over RegistrationKey for setup-GUI
	// installs; exchanged once for a permanent credential.
	EnrollmentToken string
	// ConfigFile is the JSON config path (non-secret settings only).
	// Empty selects DefaultConfigFile.
	ConfigFile string
	// StateFile persists agent_id + agent_token between restarts.
	StateFile string
	Hostname  string
	// PollInterval between server polls. Also acts as the heartbeat that
	// keeps the agent ONLINE (server marks agents stale after 3 min).
	PollInterval time.Duration
	// BrowseAddr is the loopback address of the folder browser used by
	// the dashboard picker. Empty disables it.
	BrowseAddr string
	// BrowseToken is a shared secret the dashboard must send as X-Browse-Token
	// on every browse/database-discovery request. Empty disables auth (dev only).
	BrowseToken string
	// DashboardOrigin is the allowed CORS origin for the browse server,
	// e.g. http://localhost:7540. Empty falls back to * (dev only).
	DashboardOrigin string
	// PG configures local PostgreSQL access for POSTGRES backup jobs.
	PG PgConfig
	// MSSQL configures SQL Server access for MSSQL_SERVER backup jobs.
	MSSQL MssqlConfig
	// MONGO configures MongoDB access for MONGODB backup jobs.
	MONGO MongoConfig
	// TLSSkipVerify accepts self-signed/lab certificates. Default false:
	// production agents must trust the server CA instead.
	TLSSkipVerify bool
	// JobTimeout is the maximum duration for a single backup job execution.
	// A hung pg_dump or network stall will be killed after this. Default 6h.
	JobTimeout time.Duration
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
	// Dev convenience: pick up `agent.env` (CWD = agent/) for plain `go run`
	// without a process manager. Real environment variables always win.
	_ = godotenv.Load("agent.env")

	fileCfg := loadConfigFile(configFilePath())

	poll := 30 * time.Second
	if v := os.Getenv("AGENT_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 5*time.Second {
			poll = d
		}
	} else if fileCfg.PollInterval != "" {
		if d, err := time.ParseDuration(fileCfg.PollInterval); err == nil && d >= 5*time.Second {
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
		hostname = firstNonEmpty(fileCfg.Hostname, defaultHostname())
	}
	server := os.Getenv("AGENT_SERVER")
	if server == "" {
		server = firstNonEmpty(fileCfg.ServerURL, "http://localhost:7541/vaultguard/api")
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
	pgPort := 5432
	if v := os.Getenv("AGENT_PG_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 1 && p <= 65535 {
			pgPort = p
		}
	}
	pgUser := os.Getenv("AGENT_PG_USER")
	if pgUser == "" {
		pgUser = "postgres"
	}
	mssqlServer := os.Getenv("AGENT_MSSQL_SERVER")
	if mssqlServer == "" {
		mssqlServer = "localhost"
	}
	return Config{
		Server:          server,
		RegistrationKey: os.Getenv("AGENT_REGISTRATION_KEY"),
		EnrollmentToken: os.Getenv("AGENT_ENROLLMENT_TOKEN"),
		ConfigFile:      configFilePath(),
		StateFile:       stateFile,
		Hostname:        hostname,
		PollInterval:    poll,
		BrowseAddr:      browseAddr,
		BrowseToken:     os.Getenv("AGENT_BROWSE_TOKEN"),
		DashboardOrigin: firstNonEmpty(os.Getenv("AGENT_DASHBOARD_ORIGIN"), "http://localhost:7540"),
		PG: PgConfig{
			Host:     firstNonEmpty(os.Getenv("AGENT_PG_HOST"), "localhost"),
			Port:     pgPort,
			User:     pgUser,
			Password: os.Getenv("AGENT_PG_PASSWORD"),
		},
		MSSQL: MssqlConfig{
			Server:    mssqlServer,
			User:      os.Getenv("AGENT_MSSQL_USER"),
			Password:  os.Getenv("AGENT_MSSQL_PASSWORD"),
			BackupDir: os.Getenv("AGENT_MSSQL_BACKUP_DIR"),
		},
		MONGO: MongoConfig{
			URI: firstNonEmpty(os.Getenv("AGENT_MONGO_URI"), "mongodb://localhost:27017"),
		},
		TLSSkipVerify: strings.EqualFold(os.Getenv("AGENT_TLS_SKIP_VERIFY"), "true"),
		JobTimeout:    jobTimeout(),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// --- Local config file (non-secret settings only) ---
//
// The installed agent reads C:\ProgramData\VaultGuard\Agent\config.json
// (Windows) or /etc/vaultguard/agent/config.json (Linux), written by the
// setup GUI. Database passwords and the agent token MUST NOT go here:
// credentials live only in the CredentialStore (or the legacy state file).
// Unknown/secret-looking keys are ignored, never honored.

// FileSettings mirrors config.json. Deliberately no credential fields:
// server URL, host identity, agent ID (all non-secret diagnostics).
// Exported so the setup installer writes exactly the schema the agent
// reads — a single source of truth, no drift.
type FileSettings struct {
	ServerURL    string `json:"server_url"`
	Hostname     string `json:"hostname"`
	PollInterval string `json:"poll_interval"`
	// AgentID is informational (which agent this install is). It is NOT a
	// credential — authentication needs the stored token, never this ID.
	AgentID string `json:"agent_id,omitempty"`
}

// DefaultConfigDir is the platform config/data directory for the agent.
// Mutable data lives here — never inside Program Files.
func DefaultConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		if pd := os.Getenv("ProgramData"); pd != "" {
			return filepath.Join(pd, "VaultGuard", "Agent")
		}
		return `C:\ProgramData\VaultGuard\Agent`
	case "linux":
		return "/etc/vaultguard/agent"
	default:
		if exe, err := os.Executable(); err == nil {
			return filepath.Dir(exe)
		}
		return "."
	}
}

// DefaultConfigFile is the default config.json path.
func DefaultConfigFile() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}

// CredentialDir is where the CredentialStore lives (config dir by default,
// AGENT_CREDENTIAL_DIR override for tests/portable installs).
func CredentialDir() string {
	if d := os.Getenv("AGENT_CREDENTIAL_DIR"); d != "" {
		return d
	}
	return DefaultConfigDir()
}

func configFilePath() string {
	if p := os.Getenv("AGENT_CONFIG_FILE"); p != "" {
		return p
	}
	return DefaultConfigFile()
}

// loadConfigFile reads non-secret settings; missing file = zero value
// (env/defaults apply). A file that fails to parse is reported by the
// caller-visible error return.
func loadConfigFile(path string) FileSettings {
	var fs FileSettings
	data, err := os.ReadFile(path)
	if err != nil {
		return fs
	}
	_ = json.Unmarshal(data, &fs) // unknown keys ignored; bad JSON = defaults
	return fs
}

// WriteConfigFile persists non-secret settings for service installs.
// It refuses to write anything resembling a credential.
func WriteConfigFile(path string, fs FileSettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(fs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func jobTimeout() time.Duration {
	if v := os.Getenv("AGENT_JOB_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 6 * time.Hour
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
