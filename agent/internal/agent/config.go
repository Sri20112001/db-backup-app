package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
	// StateFile persists agent_id + agent_token between restarts.
	StateFile string
	Hostname  string
	// PollInterval between server polls. Also acts as the heartbeat that
	// keeps the agent ONLINE (server marks agents stale after 3 min).
	PollInterval time.Duration
	// BrowseAddr is the loopback address of the folder browser used by
	// the dashboard picker. Empty disables it.
	BrowseAddr string
	// PG configures local PostgreSQL access for POSTGRES backup jobs.
	PG PgConfig
	// MSSQL configures SQL Server access for MSSQL_SERVER backup jobs.
	MSSQL MssqlConfig
	// MONGO configures MongoDB access for MONGODB backup jobs.
	MONGO MongoConfig
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
		server = "http://localhost:7541/vaultguard/api"
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
		StateFile:       stateFile,
		Hostname:        hostname,
		PollInterval:    poll,
		BrowseAddr:      browseAddr,
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
