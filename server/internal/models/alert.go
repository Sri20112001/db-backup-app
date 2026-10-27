package models

import (
	"time"

	"github.com/google/uuid"
)

// --- Alert ---

type AlertType string

const (
	AlertBackupFailed  AlertType = "BACKUP_FAILED"
	AlertBackupMissed  AlertType = "BACKUP_MISSED"
	AlertAgentOffline  AlertType = "AGENT_OFFLINE"
	AlertStorageLow    AlertType = "STORAGE_LOW"
	AlertRestoreFailed AlertType = "RESTORE_FAILED"
	AlertVerifyFailed  AlertType = "VERIFY_FAILED"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// SeverityRank orders severities for threshold comparisons (min-severity
// email floors, per-user prefs). Higher is more severe.
func SeverityRank(s AlertSeverity) int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	default:
		return 1
	}
}

type AlertStatus string

const (
	AlertFiring   AlertStatus = "firing"
	AlertAcked    AlertStatus = "acknowledged"
	AlertResolved AlertStatus = "resolved"
)

// DefaultSeverity maps a type to its severity when the raiser doesn't
// override it. Failures and outages are critical; integrity warnings are not.
func DefaultSeverity(t AlertType) AlertSeverity {
	switch t {
	case AlertBackupFailed, AlertBackupMissed, AlertAgentOffline, AlertRestoreFailed:
		return SeverityCritical
	default:
		return SeverityWarning
	}
}

type Alert struct {
	Base
	OrganizationID  uuid.UUID     `gorm:"type:uuid;not null;index" json:"organization_id"`
	Type            AlertType     `gorm:"not null" json:"type"`
	Severity        AlertSeverity `gorm:"not null;default:'warning'" json:"severity"`
	Status          AlertStatus   `gorm:"not null;default:'firing'" json:"status"`
	Title           string        `gorm:"not null" json:"title"`
	Message         string        `json:"message"`
	Read            bool          `gorm:"default:false" json:"read"`
	// DedupKey coalesces repeats of the same condition into one FIRING row
	// (occurrence_count/last_occurred_at). Empty = always a fresh row.
	DedupKey        *string       `json:"dedup_key,omitempty"`
	OccurrenceCount int           `gorm:"not null;default:1" json:"occurrence_count"`
	AcknowledgedBy  *uuid.UUID    `gorm:"type:uuid" json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time    `json:"acknowledged_at,omitempty"`
	ResolvedAt      *time.Time    `json:"resolved_at,omitempty"`
	LastOccurredAt  *time.Time    `json:"last_occurred_at,omitempty"`
	BackupJobID     *uuid.UUID    `gorm:"type:uuid" json:"backup_job_id,omitempty"`
	AgentID         *uuid.UUID    `gorm:"type:uuid" json:"agent_id,omitempty"`
}

// --- NotificationSettings ---
// Per-org notification routing knobs. SMTP transport itself is process-env;
// these shape who gets notified, when, and through which extra channels.

type NotificationSettings struct {
	OrganizationID      uuid.UUID     `gorm:"type:uuid;primaryKey" json:"organization_id"`
	WebhookURL          string        `gorm:"not null;default:''" json:"webhook_url"`
	WebhookEnabled      bool          `gorm:"not null;default:false" json:"webhook_enabled"`
	WebhookSecret       string        `gorm:"not null;default:''" json:"-"`
	QuietHoursStart     int           `gorm:"not null;default:-1" json:"quiet_hours_start"`
	QuietHoursEnd       int           `gorm:"not null;default:-1" json:"quiet_hours_end"`
	MinEmailSeverity    AlertSeverity `gorm:"not null;default:'warning'" json:"min_email_severity"`
	OfflineThresholdMin int           `gorm:"not null;default:3" json:"offline_threshold_minutes"`
	MissedWindowMult    int           `gorm:"not null;default:2" json:"missed_window_multiplier"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

// --- UserNotificationPrefs ---
// Per-user email preferences. Rows may not exist; defaults are permissive
// (email on, floor at info) so behavior matches the pre-prefs era.

type UserNotificationPrefs struct {
	UserID       uuid.UUID     `gorm:"type:uuid;primaryKey" json:"user_id"`
	EmailEnabled bool          `gorm:"not null;default:true" json:"email_enabled"`
	MinSeverity  AlertSeverity `gorm:"not null;default:'info'" json:"min_severity"`
	UpdatedAt    time.Time     `json:"updated_at"`
}
