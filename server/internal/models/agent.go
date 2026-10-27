package models

import (
	"time"

	"github.com/google/uuid"
)

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
	LifecyclePending = "PENDING"
	LifecycleOnline  = "ONLINE"
	LifecycleOffline = "OFFLINE"
	LifecycleRevoked = "REVOKED"
)

// OnlineThreshold is how recently last_seen_at must be for ONLINE.
// Shared by the heartbeat path and the health monitor so both agree.
const OnlineThreshold = 3 * time.Minute

type Agent struct {
	Base
	OrganizationID uuid.UUID   `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string      `gorm:"not null" json:"name"`
	// TokenHash is nil until the agent completes registration. NULLs never
	// collide in the unique index, so any number of pending (key-only) rows
	// can coexist — empty-string hashes used to violate it on the second
	// registration token (HTTP 500 from GenerateRegistrationToken).
	TokenHash      *string     `gorm:"uniqueIndex" json:"-"`
	TokenRotatedAt *time.Time  `json:"token_rotated_at"`
	Status         AgentStatus `gorm:"not null;default:'OFFLINE'" json:"status"`
	Version        string      `json:"version"`
	LastSeenAt     *time.Time  `json:"last_seen_at"`
	// RegistrationKey stores a SHA-256 hex digest of the one-time key, never
	// plaintext. Pending (never-enrolled) rows carry it; enrollment clears it.
	RegistrationKey *string    `gorm:"uniqueIndex" json:"-"`
	// Lifecycle metadata (agent milestone): inventory + revocation.
	Platform              string     `gorm:"default:''" json:"platform"`
	Architecture          string     `gorm:"default:''" json:"architecture"`
	MachineName           string     `gorm:"default:''" json:"machine_name"`
	InstalledAt           *time.Time `json:"installed_at,omitempty"`
	RevokedAt             *time.Time `json:"revoked_at,omitempty"`
	RegistrationExpiresAt *time.Time `json:"-"`
}

// Lifecycle returns the effective enrollment-aware state. ONLINE is derived
// from heartbeat recency, never trusted blindly from the stored flag (the
// health monitor reconciles the stored flag within minutes; this makes the
// API truthful immediately, including for agents that never heartbeated).
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
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
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
