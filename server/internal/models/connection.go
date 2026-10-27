package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- DatabaseConnection ---
// DatabaseConnection is a reusable infrastructure resource: connection
// metadata configured once, referenced by many backup jobs. Secrets are
// NEVER stored here in plaintext — only AES-256-GCM ciphertext (same
// ENCRYPTION_KEY envelope as storage targets), never returned by the API.
// The agent receives decrypted credentials per-claim over HTTPS, exactly
// like storage credentials; filesystem jobs leave ConnectionID NULL.
type ConnectionType string

const (
	ConnectionPostgres ConnectionType = "POSTGRES"
	ConnectionMongo    ConnectionType = "MONGODB"
	ConnectionMssql    ConnectionType = "MSSQL"
)

type ConnectionStatus string

const (
	ConnectionUnknown      ConnectionStatus = "UNKNOWN"
	ConnectionConnected    ConnectionStatus = "CONNECTED"
	ConnectionDisconnected ConnectionStatus = "DISCONNECTED"
	ConnectionError        ConnectionStatus = "ERROR"
)

type DatabaseConnection struct {
	Base
	OrganizationID uuid.UUID        `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentID        uuid.UUID        `gorm:"type:uuid;not null;index" json:"agent_id"`
	Name           string           `gorm:"not null" json:"name"`
	Type           ConnectionType   `gorm:"not null" json:"type"`
	Host           string           `gorm:"not null" json:"host"`
	Port           int              `gorm:"not null" json:"port"`
	Username       string           `gorm:"not null" json:"username"`
	// EncryptedPassword holds AES-256-GCM ciphertext (base64). json:"-" so
	// it never leaves the server except per-claim to the owning agent.
	EncryptedPassword string           `json:"-"`
	Status            ConnectionStatus `gorm:"not null;default:'UNKNOWN'" json:"status"`
	LastCheckedAt     *time.Time       `json:"last_checked_at,omitempty"`
	// DatabaseNames caches the last discovered database list (metadata,
	// not secret) so the job wizard can populate its dropdown without a
	// live agent round-trip. Refreshed on test/save.
	DatabaseNames string `gorm:"default:''" json:"-"`
	// DatabaseMetadata caches richer per-database details (JSON array of
	// DatabaseInfo: sizes, table counts) collected at Test/Refresh time.
	// Best-effort: empty when the engine doesn't expose details.
	DatabaseMetadata string `gorm:"default:''" json:"-"`
	Agent            Agent  `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

// UnknownSize marks an unavailable size/count in DatabaseInfo (-1, since 0
// is a legitimate empty-database reading).
const UnknownSize = int64(-1)

// DatabaseInfo is one cached database entry: identity plus best-effort
// details. TableCount -1 (and SizeBytes -1) mean "unknown for this engine".
type DatabaseInfo struct {
	Name       string `json:"name"`
	SizeBytes  int64  `json:"size_bytes"`
	TableCount int    `json:"table_count"`
}

// TableInfo is one table/collection inside a database (live drill-in, not
// cached). Rows -1 / SizeBytes -1 mean unknown.
type TableInfo struct {
	Schema    string `json:"schema,omitempty"`
	Name      string `json:"name"`
	Rows      int64  `json:"rows"`
	SizeBytes int64  `json:"size_bytes"`
}

// DatabaseList returns the cached database names as a slice.
func (c *DatabaseConnection) DatabaseList() []string {
	if c.DatabaseNames == "" {
		return []string{}
	}
	var out []string
	for _, n := range splitConnDBs(c.DatabaseNames) {
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func splitConnDBs(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

// MetadataList returns cached DatabaseInfo, falling back to bare names with
// unknown details when no metadata was ever collected.
func (c *DatabaseConnection) MetadataList() []DatabaseInfo {
	if strings.TrimSpace(c.DatabaseMetadata) != "" {
		var out []DatabaseInfo
		if err := json.Unmarshal([]byte(c.DatabaseMetadata), &out); err == nil && len(out) > 0 {
			return out
		}
	}
	names := c.DatabaseList()
	out := make([]DatabaseInfo, 0, len(names))
	for _, n := range names {
		out = append(out, DatabaseInfo{Name: n, SizeBytes: UnknownSize, TableCount: -1})
	}
	return out
}

// MarshalDatabaseMetadata encodes details for storage ("" when empty).
func MarshalDatabaseMetadata(infos []DatabaseInfo) string {
	if len(infos) == 0 {
		return ""
	}
	data, err := json.Marshal(infos)
	if err != nil {
		return ""
	}
	return string(data)
}

// MergeMetadataNames re-attaches cached sizes/counts to a fresh names-only
// discovery (frontend loopback sends names; details survive by name match).
func MergeMetadataNames(cached []DatabaseInfo, names []string) []DatabaseInfo {
	byName := map[string]DatabaseInfo{}
	for _, d := range cached {
		byName[d.Name] = d
	}
	out := make([]DatabaseInfo, 0, len(names))
	for _, n := range names {
		if d, ok := byName[n]; ok {
			out = append(out, d)
		} else {
			out = append(out, DatabaseInfo{Name: n, SizeBytes: UnknownSize, TableCount: -1})
		}
	}
	return out
}
