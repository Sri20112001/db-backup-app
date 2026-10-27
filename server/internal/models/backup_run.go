package models

import (
	"time"

	"github.com/google/uuid"
)

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
	ScheduledFor    *time.Time      `json:"scheduled_for,omitempty"`
	// CancelRequested is set by Cancel; the agent honors it at stage
	// boundaries and reports CANCELLED itself. PENDING runs flip to
	// CANCELLED immediately since no agent owns them yet.
	CancelRequested bool            `gorm:"default:false" json:"cancel_requested"`
	// Retry bookkeeping: retries are NEW runs (terminal states never
	// transition out), linked here for observability. RetryOfRunID is the
	// failed run this run retries; RetryAttempt counts refires (0 = original).
	RetryOfRunID    *uuid.UUID      `gorm:"type:uuid;index" json:"retry_of_run_id,omitempty"`
	RetryAttempt    int             `gorm:"default:0" json:"retry_attempt"`
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

// --- BackupArtifact ---

type BackupArtifact struct {
	Base
	BackupRunID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"backup_run_id"`
	Name               string     `gorm:"not null" json:"name"`
	Size               int64      `json:"size"`
	Checksum           string     `json:"checksum"`
	StoragePath        string     `json:"storage_path"`
	// StorageTargetID pins the exact target this artifact was written to,
	// even if the job's target is changed or deleted later. Nullable at
	// the DB level so orphaned historical rows can never block a
	// migration; the API always fills it for new artifacts.
	StorageTargetID    uuid.UUID  `gorm:"type:uuid;index" json:"storage_target_id"`
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
	// Idempotency key is (artifact_id, index): chunk N of one artifact.
	// Both columns share one composite unique index — a single-column
	// unique on index alone would collide chunk N across different
	// artifacts of the same run.
	ArtifactID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_chunk_artifact_index" json:"artifact_id"`
	Index       int       `gorm:"not null;uniqueIndex:idx_chunk_artifact_index" json:"index"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	StoragePath string    `json:"storage_path"`
	Uploaded    bool      `gorm:"default:false" json:"uploaded"`
}
