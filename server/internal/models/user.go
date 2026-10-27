package models

import (
	"time"

	"github.com/google/uuid"
)

// --- User ---

type User struct {
	Base
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Name         string `json:"name"`
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
