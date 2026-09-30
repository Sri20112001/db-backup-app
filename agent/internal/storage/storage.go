// Package storage abstracts where finished backup payloads live.
//
// The backup runner only knows this interface:
//
//	storage.Put(...)  → upload + verify
//	storage.Get(...)  → download for restore / verification
//	storage.Head(...) → existence + size + checksum metadata
//	storage.Delete(...) → cleanup / retention
//
// Providers: LocalStorage (filesystem target), S3Storage (any
// S3-compatible API: AWS, R2, Wasabi, MinIO). SMB is a future provider.
package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

// Object metadata written with every artifact. The SHA-256 is VaultGuard's
// canonical checksum (S3 ETags are NOT content hashes for multipart
// uploads), so verification compares this value — provider-independent.
const (
	MetaChecksumSHA256 = "vaultguard-checksum-sha256"
	MetaRunID          = "vaultguard-run-id"
	MetaArtifactName   = "vaultguard-artifact-name"
	MetaVersion        = "vaultguard-format-version"
	FormatVersion      = "1"
)

// ObjectInfo describes a stored object. Location is provider-addressable:
// a filesystem path for LocalStorage, the object key for S3Storage.
type ObjectInfo struct {
	Key            string
	Size           int64
	ETag           string
	ChecksumSHA256 string
	Location       string
	Metadata       map[string]string
}

// PutOptions tunes an upload.
type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

// Storage is the provider contract. All methods honor ctx cancellation.
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) (*ObjectInfo, error)
	Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Head(ctx context.Context, key string) (*ObjectInfo, error)
}

// BuildKey lays out artifact keys as
// orgs/{orgID}/jobs/{jobID}/runs/{runID}/{filename}. UUID segments (never
// display names) keep keys stable when jobs are renamed, and the org prefix
// isolates tenants sharing one bucket. Never store full https:// URLs in
// the database — target + key resolve to the object.
func BuildKey(orgID, jobID, runID, filename string) string {
	return path.Join("orgs", orgID, "jobs", jobID, "runs", runID, filename)
}

var safeSegment = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// ArtifactFilename derives a stable, filesystem- and S3-safe artifact name
// from the job name, run id, and payload type. Compound extensions
// (.tar.gz, .sql.gz, …) are preserved; encrypted payloads keep .enc.
func ArtifactFilename(jobName, runID, archivePath string) string {
	base := strings.ToLower(safeSegment.ReplaceAllString(jobName, "_"))
	if base == "" {
		base = "backup"
	}
	// Keep compound extensions (.tar.gz, .sql.gz) when present. Encrypted
	// payloads carry a trailing .enc — look past it at the real extension.
	probe := strings.TrimSuffix(archivePath, ".enc")
	ext := ".tar"
	switch {
	case strings.HasSuffix(probe, ".tar.gz"):
		ext = ".tar.gz"
	case strings.HasSuffix(probe, ".sql.gz"):
		ext = ".sql.gz"
	case strings.HasSuffix(probe, ".sql"):
		ext = ".sql"
	case strings.HasSuffix(probe, ".bak"):
		ext = ".bak"
	case strings.HasSuffix(probe, ".archive.gz"):
		ext = ".archive.gz"
	case strings.HasSuffix(probe, ".json.gz"):
		ext = ".json.gz"
	case strings.HasSuffix(probe, ".json"):
		ext = ".json"
	case strings.HasSuffix(probe, ".csv.gz"):
		ext = ".csv.gz"
	case strings.HasSuffix(probe, ".csv"):
		ext = ".csv"
	case strings.HasSuffix(probe, ".vgm"):
		ext = ".vgm"
	}
	if strings.HasSuffix(archivePath, ".enc") {
		ext += ".enc"
	}
	return fmt.Sprintf("%s_%s%s", base, runID, ext)
}

// ContentTypeForName guesses an upload content type; unknown payloads are
// opaque bytes.
func ContentTypeForName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".gz"):
		return "application/gzip"
	case strings.HasSuffix(lower, ".tar"):
		return "application/x-tar"
	case strings.HasSuffix(lower, ".sql"):
		return "application/sql"
	case strings.HasSuffix(lower, ".json"):
		return "application/json"
	case strings.HasSuffix(lower, ".csv"):
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
