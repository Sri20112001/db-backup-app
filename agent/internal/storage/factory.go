package storage

import (
	"fmt"
	"strings"
)

// S3Config carries everything S3Storage needs. It is assembled by the
// backup runner from the server's claim response (credentials decrypted
// server-side, delivered per-claim over HTTPS).
type S3Config struct {
	Bucket       string
	Region       string
	Endpoint     string
	UsePathStyle bool
	AccessKey    string
	SecretKey    string
}

// ForJob builds the provider for one backup job. Unknown types fail here —
// before the backup runs, never after the payload is produced.
func ForJob(storageType, root string, s3 *S3Config) (Storage, error) {
	switch strings.ToUpper(strings.TrimSpace(storageType)) {
	case "", "LOCAL":
		return NewLocalStorage(root)
	case "S3":
		return NewS3Storage(s3)
	default:
		return nil, fmt.Errorf("unsupported storage type %q (supported: LOCAL, S3)", storageType)
	}
}
