package models

import (
	"github.com/google/uuid"
)

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
	// UsePathStyle selects path-style addressing (http://host/bucket/key,
	// MinIO-friendly) over virtual-hosted style (AWS/R2/Wasabi).
	UsePathStyle       bool        `gorm:"default:false" json:"use_path_style"`
	EncryptedAccessKey string      `json:"-"`
	EncryptedSecretKey string      `json:"-"`
	Path               string      `json:"path"`
	// HasCredentials is transient (never stored): tells the UI whether
	// credentials exist without ever returning them.
	HasCredentials     bool        `gorm:"-" json:"has_credentials"`
}

// --- S3Region ---

// S3Region is reference data for the storage-target region picker: AWS
// region codes plus S3-compatible presets. Seeded by migration 000004;
// admins can add private endpoints (air-gapped clones) via the API without
// a redeploy. StorageTarget.Region stores the code as free text, so unknown
// future regions keep working even before anyone inserts them here.
type S3Region struct {
	Base
	Code     string `gorm:"uniqueIndex;not null" json:"code"`
	Name     string `gorm:"not null" json:"name"`
	Provider string `gorm:"not null;default:'AWS'" json:"provider"`
	Endpoint string `json:"endpoint,omitempty"`
	IsSystem bool   `gorm:"default:true" json:"is_system"`
	Active   bool   `gorm:"default:true" json:"active"`
}
