package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Base ---

type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// --- Organization ---

type Organization struct {
	Base
	Name      string `gorm:"not null" json:"name"`
	Slug      string `gorm:"uniqueIndex;not null" json:"slug"`
	UserLimit int    `gorm:"not null;default:10" json:"user_limit"`
	Users     []OrganizationMember `gorm:"foreignKey:OrganizationID" json:"-"`
}

// --- User ---

type User struct {
	Base
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Name         string `json:"name"`
}

// --- OrganizationMember ---

type MemberRole string

const (
	RoleOwner    MemberRole = "OWNER"
	RoleAdmin    MemberRole = "ADMIN"
	RoleOperator MemberRole = "OPERATOR"
	RoleViewer   MemberRole = "VIEWER"
)

type OrganizationMember struct {
	Base
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	Role           MemberRole   `gorm:"not null;default:'VIEWER'" json:"role"`
	Organization   Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	User           User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// --- Agent ---

type AgentStatus string

const (
	AgentOnline  AgentStatus = "ONLINE"
	AgentOffline AgentStatus = "OFFLINE"
)

// AgentLifecycle is the effective enrollment-aware state exposed by the API
// (transient `lifecycle` field, never stored): PENDING = created but never
// enrolled (no permanent credential yet); REVOKED = administratively
// revoked (authentication refused); otherwise the ONLINE/OFFLINE heartbeat
// state. REGISTERED corresponds to enrolled (InstalledAt set, not pending).
const (
	LifecyclePending  = "PENDING"
	LifecycleOnline   = "ONLINE"
	LifecycleOffline  = "OFFLINE"
	LifecycleRevoked  = "REVOKED"
)

// OnlineThreshold is how recently last_seen_at must be for ONLINE.
// Shared by the heartbeat path and the health monitor so both agree.
const OnlineThreshold = 3 * time.Minute

type Agent struct {
	Base
	OrganizationID  uuid.UUID   `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name            string      `gorm:"not null" json:"name"`
	// TokenHash is nil until the agent completes registration. NULLs never
	// collide in the unique index, so any number of pending (key-only) rows
	// can coexist — empty-string hashes used to violate it on the second
	// registration token (HTTP 500 from GenerateRegistrationToken).
	TokenHash       *string     `gorm:"uniqueIndex" json:"-"`
	TokenRotatedAt  *time.Time  `json:"token_rotated_at"`
	Status          AgentStatus `gorm:"not null;default:'OFFLINE'" json:"status"`
	Version         string      `json:"version"`
	LastSeenAt      *time.Time  `json:"last_seen_at"`
	// RegistrationKey stores a SHA-256 hex digest of the one-time key, never
	// plaintext. Pending (never-enrolled) rows carry it; enrollment clears it.
	RegistrationKey *string     `gorm:"uniqueIndex" json:"-"`
	// Lifecycle metadata (agent milestone): inventory + revocation.
	Platform        string      `gorm:"default:''" json:"platform"`
	Architecture    string      `gorm:"default:''" json:"architecture"`
	MachineName     string      `gorm:"default:''" json:"machine_name"`
	InstalledAt     *time.Time  `json:"installed_at,omitempty"`
	RevokedAt       *time.Time  `json:"revoked_at,omitempty"`
	RegistrationExpiresAt *time.Time `json:"-"`
}

// Lifecycle returns the effective enrollment-aware state. ONLINE is derived
// from heartbeat recency, never trusted blindly from the stored flag.
func (a *Agent) Lifecycle() string {
	if a.RevokedAt != nil {
		return LifecycleRevoked
	}
	if a.TokenHash == nil {
		return LifecyclePending
	}
	if a.LastSeenAt != nil && time.Since(*a.LastSeenAt) < OnlineThreshold {
		return LifecycleOnline
	}
	if a.Status == AgentOnline {
		return LifecycleOnline
	}
	return LifecycleOffline
}

// --- EnrollmentToken ---
// EnrollmentToken is a short-lived single-use token for agent enrollment.
// Only the SHA-256 hex digest is stored; plaintext is shown once at
// creation. Optionally linked to a pre-created pending Agent row.
type EnrollmentToken struct {
	Base
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentID        *uuid.UUID `gorm:"type:uuid;index" json:"agent_id,omitempty"`
	TokenHash      string     `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt      time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"-"`
}

// --- Machine ---

type Machine struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentID        uuid.UUID `gorm:"type:uuid;not null;index" json:"agent_id"`
	Hostname       string    `gorm:"not null" json:"hostname"`
	OS             string    `json:"os"`
	IPAddress      string    `json:"ip_address"`
	Agent          Agent     `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

// --- StorageTarget ---

type StorageType string

const (
	StorageS3    StorageType = "S3"
	StorageLocal StorageType = "LOCAL"
	StorageSMB   StorageType = "SMB"
)

type StorageTarget struct {
	Base
	OrganizationID     uuid.UUID   `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name               string      `gorm:"not null" json:"name"`
	Type               StorageType `gorm:"not null" json:"type"`
	Bucket             string    `json:"bucket"`
	Region             string    `json:"region"`
	Endpoint           string    `json:"endpoint"`
	// UsePathStyle selects path-style addressing (http://host/bucket/key,
	// MinIO-friendly) over virtual-hosted style (AWS/R2/Wasabi).
	UsePathStyle       bool      `gorm:"default:false" json:"use_path_style"`
	EncryptedAccessKey string    `json:"-"`
	EncryptedSecretKey string    `json:"-"`
	Path               string    `json:"path"`
	// HasCredentials is transient (never stored): tells the UI whether
	// credentials exist without ever returning them.
	HasCredentials     bool      `gorm:"-" json:"has_credentials"`
}

// --- S3Region ---

// S3Region is reference data for the storage-target region picker: AWS
// region codes plus S3-compatible presets. Seeded by migration 000004;
// admins can add private endpoints (air-gapped clones) via the API without
// a redeploy. StorageTarget.Region stores the code as free text, so unknown
// future regions keep working even before anyone inserts them here.
type S3Region struct {
	Base
	Code      string `gorm:"uniqueIndex;not null" json:"code"`
	Name      string `gorm:"not null" json:"name"`
	Provider  string `gorm:"not null;default:'AWS'" json:"provider"`
	Endpoint  string `json:"endpoint,omitempty"`
	IsSystem  bool   `gorm:"default:true" json:"is_system"`
	Active    bool   `gorm:"default:true" json:"active"`
}

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
	Agent         Agent  `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
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

// --- BackupJob ---

type BackupSourceType string

const (
	SourceFilesystem BackupSourceType = "FILESYSTEM"
	SourceSQLServer  BackupSourceType = "SQL_SERVER"
	// SourceMssqlServer is what the dashboard sends for SQL Server jobs;
	// both spellings are accepted wherever source types are matched.
	SourceMssqlServer BackupSourceType = "MSSQL_SERVER"
	SourceDBF        BackupSourceType = "DBF"
	SourcePostgres   BackupSourceType = "POSTGRES"
	SourceMongo      BackupSourceType = "MONGODB"
)

type BackupMode string

const (
	ModeNormal     BackupMode = "NORMAL"
	ModeCompressed BackupMode = "COMPRESSED"
)

type BackupJob struct {
	Base
	OrganizationID  uuid.UUID        `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"agent_id"`
	StorageTargetID uuid.UUID        `gorm:"type:uuid;not null;index" json:"storage_target_id"`
	Name            string           `gorm:"not null" json:"name"`
	SourceType      BackupSourceType `gorm:"not null" json:"source_type"`
	// ConnectionID references a saved DatabaseConnection for database jobs.
	// NULL for filesystem jobs and for legacy jobs created before the
	// connections feature (backward compatible).
	ConnectionID    *uuid.UUID       `gorm:"type:uuid;index" json:"connection_id,omitempty"`
	SourcePath      string           `json:"source_path"`
	SourceDatabase  string           `json:"source_database"`
	IncludePatterns string           `json:"include_patterns"`
	ExcludePatterns string           `json:"exclude_patterns"`
	// ExportFormat selects the MongoDB payload shape: ARCHIVE (default,
	// mongodump-compatible binary), JSON (one EJSON doc per line), or CSV
	// (flattened, header row; values restore as strings). Ignored by other
	// source types.
	ExportFormat    string           `gorm:"default:'ARCHIVE'" json:"export_format"`
	Mode            BackupMode       `gorm:"not null;default:'NORMAL'" json:"mode"`
	Encrypted       bool             `gorm:"default:false" json:"encrypted"`
	RetentionDays   int              `gorm:"default:30" json:"retention_days"`
	Enabled         bool             `gorm:"default:true" json:"enabled"`
	// SLA / RPO / RTO targets (0 = not configured).
	// SLATargetMinutes: max minutes between schedule fire and backup completion.
	// RPOTargetMinutes: max acceptable data-loss window (drives missed-backup alerting).
	// RTOTargetMinutes: max acceptable restore time (tracked against RestoreJob.DurationSeconds).
	SLATargetMinutes int             `gorm:"default:0" json:"sla_target_minutes"`
	RPOTargetMinutes int             `gorm:"default:0" json:"rpo_target_minutes"`
	RTOTargetMinutes int             `gorm:"default:0" json:"rto_target_minutes"`
	Schedule        *BackupSchedule  `gorm:"foreignKey:BackupJobID" json:"schedule,omitempty"`
	Agent           Agent            `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	StorageTarget   StorageTarget    `gorm:"foreignKey:StorageTargetID" json:"storage_target,omitempty"`
	Connection      *DatabaseConnection `gorm:"foreignKey:ConnectionID" json:"connection,omitempty"`
}

// --- BackupSchedule ---

type BackupSchedule struct {
	Base
	BackupJobID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"backup_job_id"`
	CronExpr    string    `gorm:"not null" json:"cron_expr"`
	Timezone    string    `gorm:"default:'UTC'" json:"timezone"`
}

// --- BackupRun ---

type BackupRunStatus string

const (
	RunPending   BackupRunStatus = "PENDING"
	RunRunning   BackupRunStatus = "RUNNING"
	RunUploading BackupRunStatus = "UPLOADING"
	RunVerifying BackupRunStatus = "VERIFYING"
	RunCompleted BackupRunStatus = "COMPLETED"
	RunFailed    BackupRunStatus = "FAILED"
	RunCancelled BackupRunStatus = "CANCELLED"
)

type BackupRun struct {
	Base
	OrganizationID  uuid.UUID       `gorm:"type:uuid;not null;index" json:"organization_id"`
	BackupJobID     uuid.UUID       `gorm:"type:uuid;not null;index" json:"backup_job_id"`
	AgentID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"agent_id"`
	StorageTargetID uuid.UUID       `gorm:"type:uuid;not null" json:"storage_target_id"`
	Status          BackupRunStatus `gorm:"not null;default:'PENDING'" json:"status"`
	// ScheduledFor is the cron boundary this run was fired for. Scheduler
	// inserts carry it (UNIQUE with backup_job_id, so two racing schedulers
	// — or a restart replay — can never double-fire the same boundary);
	// ad-hoc RunNow runs leave it NULL (NULLs never collide).
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	// CancelRequested is set by Cancel; the agent honors it at stage
	// boundaries and reports CANCELLED itself. PENDING runs flip to
	// CANCELLED immediately since no agent owns them yet.
	CancelRequested bool             `gorm:"default:false" json:"cancel_requested"`
	StartedAt       *time.Time      `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at"`
	BytesRead       int64           `json:"bytes_read"`
	BytesCompressed int64           `json:"bytes_compressed"`
	BytesUploaded   int64           `json:"bytes_uploaded"`
	DurationSeconds int64           `json:"duration_seconds"`
	SourceType      string          `json:"source_type"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	// FailureCategory classifies the error for structured alerting and UI display.
	FailureCategory string          `gorm:"default:''" json:"failure_category,omitempty"`
	StoragePath     string          `json:"storage_path"`
	Checksum        string          `json:"checksum"`
	// DataKeyEncrypted holds the per-backup AES-256 data key, envelope-
	// encrypted with the server ENCRYPTION_KEY (same as storage credentials).
	// The server never sees plaintext backup data, only the wrapped key.
	DataKeyEncrypted string         `json:"-"`
	BackupJob       BackupJob       `gorm:"foreignKey:BackupJobID" json:"backup_job,omitempty"`
}

// CanTransition returns true if moving from → to is a valid state transition.
func CanTransition(from, to BackupRunStatus) bool {
	switch from {
	case RunPending:
		return to == RunRunning || to == RunCancelled
	case RunRunning:
		return to == RunUploading || to == RunFailed || to == RunCancelled
	case RunUploading:
		return to == RunVerifying || to == RunFailed || to == RunCancelled
	case RunVerifying:
		return to == RunCompleted || to == RunFailed || to == RunCancelled
	}
	return false
}



type BackupArtifact struct {
	Base
	BackupRunID uuid.UUID `gorm:"type:uuid;not null;index" json:"backup_run_id"`
	Name        string    `gorm:"not null" json:"name"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	StoragePath string    `json:"storage_path"`
	// StorageTargetID pins the exact target this artifact was written to,
	// even if the job's target is changed or deleted later. Nullable at
	// the DB level so orphaned historical rows can never block a
	// migration; the API always fills it for new artifacts.
	StorageTargetID uuid.UUID `gorm:"type:uuid;index" json:"storage_target_id"`
	// Verification tracks HEAD/checksum confirmation (and later, recovery
	// testing): PENDING → VERIFIED | FAILED.
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	VerificationStatus string     `gorm:"default:'PENDING'" json:"verification_status"`
}

const (
	ArtifactUnverified = "PENDING"
	ArtifactVerified   = "VERIFIED"
	ArtifactVerifyFail = "FAILED"
)

// --- BackupChunk ---

type BackupChunk struct {
	Base
	ArtifactID  uuid.UUID `gorm:"type:uuid;not null;index" json:"artifact_id"`
	Index       int       `gorm:"not null;uniqueIndex:idx_chunk_artifact_index" json:"index"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	StoragePath string    `json:"storage_path"`
	Uploaded    bool      `gorm:"default:false" json:"uploaded"`
}

// --- RestoreJob ---

type RestoreStatus string

const (
	RestorePending   RestoreStatus = "PENDING"
	RestoreRunning   RestoreStatus = "RUNNING"
	RestoreCompleted RestoreStatus = "COMPLETED"
	RestoreFailed    RestoreStatus = "FAILED"
)

type RestoreJob struct {
	Base
	OrganizationID    uuid.UUID     `gorm:"type:uuid;not null;index" json:"organization_id"`
	BackupRunID       uuid.UUID     `gorm:"type:uuid;not null;index" json:"backup_run_id"`
	AgentID           uuid.UUID     `gorm:"type:uuid;not null;index" json:"agent_id"`
	// ConnectionID pins the saved connection for database restores, copied
	// from the source backup job at creation. NULL for filesystem restores
	// and legacy rows (agent env fallback applies).
	ConnectionID      *uuid.UUID    `gorm:"type:uuid;index" json:"connection_id,omitempty"`
	Status            RestoreStatus `gorm:"not null;default:'PENDING'" json:"status"`
	DestinationPath   string        `json:"destination_path"`
	TargetDatabase    string        `json:"target_database"`
	StartedAt         *time.Time    `json:"started_at"`
	CompletedAt       *time.Time    `json:"completed_at"`
	DurationSeconds   int64         `json:"duration_seconds"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	BackupRun         BackupRun     `gorm:"foreignKey:BackupRunID" json:"backup_run,omitempty"`
	Connection        *DatabaseConnection `gorm:"foreignKey:ConnectionID" json:"connection,omitempty"`
}

// --- BackupSizeBaseline ---
// Tracks rolling average backup size per job for anomaly detection.
// Updated by the health monitor after each completed run.

type BackupSizeBaseline struct {
	Base
	BackupJobID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"backup_job_id"`
	AvgBytes      int64     `gorm:"not null;default:0" json:"avg_bytes"`
	SampleCount   int       `gorm:"not null;default:0" json:"sample_count"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// --- Alert ---

type AlertType string

const (
	AlertBackupFailed    AlertType = "BACKUP_FAILED"
	AlertBackupMissed    AlertType = "BACKUP_MISSED"
	AlertAgentOffline    AlertType = "AGENT_OFFLINE"
	AlertStorageLow      AlertType = "STORAGE_LOW"
	AlertRestoreFailed   AlertType = "RESTORE_FAILED"
	AlertVerifyFailed    AlertType = "VERIFY_FAILED"
)

type Alert struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	Type           AlertType `gorm:"not null" json:"type"`
	Title          string    `gorm:"not null" json:"title"`
	Message        string    `json:"message"`
	Read           bool      `gorm:"default:false" json:"read"`
	BackupJobID    *uuid.UUID `gorm:"type:uuid" json:"backup_job_id,omitempty"`
	AgentID        *uuid.UUID `gorm:"type:uuid" json:"agent_id,omitempty"`
}

// --- AuditLog ---

type AuditLog struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Action         string    `gorm:"not null" json:"action"`
	Resource       string    `json:"resource"`
	ResourceID     string    `json:"resource_id"`
	IPAddress      string    `json:"ip_address"`
}

// --- RefreshToken ---

type RefreshToken struct {
	Base
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	FamilyID  uuid.UUID `gorm:"type:uuid;not null;index" json:"-"` // all tokens in one login session share a family
	Token     string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `gorm:"default:false" json:"-"`
}
