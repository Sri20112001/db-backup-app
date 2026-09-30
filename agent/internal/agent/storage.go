package agent

import (
	"os"
	"regexp"
)

var safeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// payloadSize returns a payload file's size for the post-backup data check.
func payloadSize(path string) (int64, error) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

// NOTE: artifact uploads moved to internal/storage (Storage provider
// interface + LocalStorage/S3Storage). This file keeps the shared helpers
// still used by the runner (payloadSize) and mssql backups (safeName).
