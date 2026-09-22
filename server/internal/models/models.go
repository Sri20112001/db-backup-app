package models

import (
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
	Name  string `gorm:"not null" json:"name"`
	Slug  string `gorm:"uniqueIndex;not null" json:"slug"`
	Users []OrganizationMember `gorm:"foreignKey:OrganizationID" json:"-"`
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

type Agent struct {
	Base
	OrganizationID  uuid.UUID   `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name            string      `gorm:"not null" json:"name"`
	Token           *string     `gorm:"uniqueIndex" json:"-"`
	Status          AgentStatus `gorm:"not null;default:'OFFLINE'" json:"status"`
	Version         string      `json:"version"`
	LastSeenAt      *time.Time  `json:"last_seen_at"`
	RegistrationKey *string     `gorm:"uniqueIndex" json:"-"`
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
	Bucket             string      `json:"bucket"`
	Region             string      `json:"region"`
	Endpoint           string      `json:"endpoint"`
	EncryptedAccessKey string      `json:"-"`
	EncryptedSecretKey string      `json:"-"`
	Path               string      `json:"path"`
}

// --- BackupJob ---

type BackupSourceType string

const (
	SourceFilesystem BackupSourceType = "FILESYSTEM"
	SourceSQLServer  BackupSourceType = "SQL_SERVER"
	SourceDBF        BackupSourceType = "DBF"
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
	SourcePath      string           `json:"source_path"`
	SourceDatabase  string           `json:"source_database"`
	IncludePatterns string           `json:"include_patterns"`
	ExcludePatterns string           `json:"exclude_patterns"`
	Mode            BackupMode       `gorm:"not null;default:'NORMAL'" json:"mode"`
	Encrypted       bool             `gorm:"default:false" json:"encrypted"`
	RetentionDays   int              `gorm:"default:30" json:"retention_days"`
	Enabled         bool             `gorm:"default:true" json:"enabled"`
	Schedule        *BackupSchedule  `gorm:"foreignKey:BackupJobID" json:"schedule,omitempty"`
	Agent           Agent            `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	StorageTarget   StorageTarget    `gorm:"foreignKey:StorageTargetID" json:"storage_target,omitempty"`
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
	StartedAt       *time.Time      `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at"`
	BytesRead       int64           `json:"bytes_read"`
	BytesCompressed int64           `json:"bytes_compressed"`
	BytesUploaded   int64           `json:"bytes_uploaded"`
	DurationSeconds int64           `json:"duration_seconds"`
	SourceType      string          `json:"source_type"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	StoragePath     string          `json:"storage_path"`
	Checksum        string          `json:"checksum"`
	BackupJob       BackupJob       `gorm:"foreignKey:BackupJobID" json:"backup_job,omitempty"`
}

// --- BackupArtifact ---

type BackupArtifact struct {
	Base
	BackupRunID uuid.UUID `gorm:"type:uuid;not null;index" json:"backup_run_id"`
	Name        string    `gorm:"not null" json:"name"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	StoragePath string    `json:"storage_path"`
}

// --- BackupChunk ---

type BackupChunk struct {
	Base
	ArtifactID  uuid.UUID `gorm:"type:uuid;not null;index" json:"artifact_id"`
	Index       int       `gorm:"not null" json:"index"`
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
	Status            RestoreStatus `gorm:"not null;default:'PENDING'" json:"status"`
	DestinationPath   string        `json:"destination_path"`
	TargetDatabase    string        `json:"target_database"`
	StartedAt         *time.Time    `json:"started_at"`
	CompletedAt       *time.Time    `json:"completed_at"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	BackupRun         BackupRun     `gorm:"foreignKey:BackupRunID" json:"backup_run,omitempty"`
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
	Token     string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `gorm:"default:false" json:"-"`
}
