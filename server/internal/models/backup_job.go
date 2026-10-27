package models

import (
	"time"

	"github.com/google/uuid"
)

// --- BackupJob ---

type BackupSourceType string

const (
	SourceFilesystem BackupSourceType = "FILESYSTEM"
	SourceSQLServer  BackupSourceType = "SQL_SERVER"
	// SourceMssqlServer is what the dashboard sends for SQL Server jobs;
	// both spellings are accepted wherever source types are matched.
	SourceMssqlServer BackupSourceType = "MSSQL_SERVER"
	SourceDBF         BackupSourceType = "DBF"
	SourcePostgres    BackupSourceType = "POSTGRES"
	SourceMongo       BackupSourceType = "MONGODB"
)

type BackupMode string

const (
	ModeNormal     BackupMode = "NORMAL"
	ModeCompressed BackupMode = "COMPRESSED"
)

// Backup strategy. FULL is the only strategy the backup engines implement;
// the column exists so INCREMENTAL/DIFFERENTIAL can land later WITHOUT
// renaming full backups into something they are not (validation rejects
// anything but FULL until the engines implement the semantics).
const StrategyFull = "FULL"

type BackupJob struct {
	Base
	OrganizationID   uuid.UUID           `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentID          uuid.UUID           `gorm:"type:uuid;not null;index" json:"agent_id"`
	StorageTargetID  uuid.UUID           `gorm:"type:uuid;not null;index" json:"storage_target_id"`
	// PolicyID optionally attaches a BackupPolicy. NULL means "job-inline
	// settings" — every pre-policy job keeps working with zero migration.
	// Effective settings resolve policy-first, job-inline fallback (see
	// EffectivePolicy).
	PolicyID         *uuid.UUID          `gorm:"type:uuid;index" json:"policy_id,omitempty"`
	Name             string              `gorm:"not null" json:"name"`
	SourceType       BackupSourceType    `gorm:"not null" json:"source_type"`
	// ConnectionID references a saved DatabaseConnection for database jobs.
	// NULL for filesystem jobs and for legacy jobs created before the
	// connections feature (backward compatible).
	ConnectionID     *uuid.UUID          `gorm:"type:uuid;index" json:"connection_id,omitempty"`
	SourcePath       string              `json:"source_path"`
	SourceDatabase   string              `json:"source_database"`
	IncludePatterns  string              `json:"include_patterns"`
	ExcludePatterns  string              `json:"exclude_patterns"`
	// ExportFormat selects the MongoDB payload shape: ARCHIVE (default,
	// mongodump-compatible binary), JSON (one EJSON doc per line), or CSV
	// (flattened, header row; values restore as strings). Ignored by other
	// source types.
	ExportFormat     string              `gorm:"default:'ARCHIVE'" json:"export_format"`
	Mode             BackupMode          `gorm:"not null;default:'NORMAL'" json:"mode"`
	Encrypted        bool                `gorm:"default:false" json:"encrypted"`
	RetentionDays    int                 `gorm:"default:30" json:"retention_days"`
	Enabled          bool                `gorm:"default:true" json:"enabled"`
	// SLA / RPO / RTO targets (0 = not configured).
	// SLATargetMinutes: max minutes between schedule fire and backup completion.
	// RPOTargetMinutes: max acceptable data-loss window (drives missed-backup alerting).
	// RTOTargetMinutes: max acceptable restore time (tracked against RestoreJob.DurationSeconds).
	SLATargetMinutes int                 `gorm:"default:0" json:"sla_target_minutes"`
	RPOTargetMinutes int                 `gorm:"default:0" json:"rpo_target_minutes"`
	RTOTargetMinutes int                 `gorm:"default:0" json:"rto_target_minutes"`
	Schedule         *BackupSchedule     `gorm:"foreignKey:BackupJobID" json:"schedule,omitempty"`
	Policy           *BackupPolicy       `gorm:"foreignKey:PolicyID" json:"policy,omitempty"`
	Agent            Agent               `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	StorageTarget    StorageTarget       `gorm:"foreignKey:StorageTargetID" json:"storage_target,omitempty"`
	Connection       *DatabaseConnection `gorm:"foreignKey:ConnectionID" json:"connection,omitempty"`
}

// --- BackupSchedule ---

type BackupSchedule struct {
	Base
	BackupJobID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"backup_job_id"`
	CronExpr    string    `gorm:"not null" json:"cron_expr"`
	Timezone    string    `gorm:"default:'UTC'" json:"timezone"`
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
