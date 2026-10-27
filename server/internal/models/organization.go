package models

import (
	"github.com/google/uuid"
)

// --- Organization ---

type Organization struct {
	Base
	Name      string               `gorm:"not null" json:"name"`
	Slug      string               `gorm:"uniqueIndex;not null" json:"slug"`
	UserLimit int                  `gorm:"not null;default:10" json:"user_limit"`
	Users     []OrganizationMember `gorm:"foreignKey:OrganizationID" json:"-"`
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
