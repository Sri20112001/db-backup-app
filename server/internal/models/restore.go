package models

import (
	"time"

	"github.com/google/uuid"
)

// --- RestoreJob ---

type RestoreStatus string

const (
	RestorePending   RestoreStatus = "PENDING"
	RestoreRunning   RestoreStatus = "RUNNING"
	RestoreCompleted RestoreStatus = "COMPLETED"
	RestoreFailed    RestoreStatus = "FAILED"
)

// CanRestoreTransition returns true if moving from → to is a valid restore
// state transition. Terminal states never transition out; a repeated report
// of the current state is idempotent at the handler layer (same payload
// echoed back), not a transition.
func CanRestoreTransition(from, to RestoreStatus) bool {
	switch from {
	case RestorePending:
		return to == RestoreRunning || to == RestoreFailed
	case RestoreRunning:
		return to == RestoreCompleted || to == RestoreFailed
	}
	return false
}

type RestoreJob struct {
	Base
	OrganizationID  uuid.UUID           `gorm:"type:uuid;not null;index" json:"organization_id"`
	BackupRunID     uuid.UUID           `gorm:"type:uuid;not null;index" json:"backup_run_id"`
	AgentID         uuid.UUID           `gorm:"type:uuid;not null;index" json:"agent_id"`
	// ConnectionID pins the saved connection for database restores, copied
	// from the source backup job at creation. NULL for filesystem restores
	// and legacy rows (agent env fallback applies).
	ConnectionID    *uuid.UUID          `gorm:"type:uuid;index" json:"connection_id,omitempty"`
	Status          RestoreStatus       `gorm:"not null;default:'PENDING'" json:"status"`
	DestinationPath string              `json:"destination_path"`
	TargetDatabase  string              `json:"target_database"`
	StartedAt       *time.Time          `json:"started_at"`
	CompletedAt     *time.Time          `json:"completed_at"`
	DurationSeconds int64               `json:"duration_seconds"`
	ErrorMessage    string              `json:"error_message,omitempty"`
	BackupRun       BackupRun           `gorm:"foreignKey:BackupRunID" json:"backup_run,omitempty"`
	Connection      *DatabaseConnection `gorm:"foreignKey:ConnectionID" json:"connection,omitempty"`
}
