package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// StoreLocal copies the finished archive into a LOCAL storage target
// directory and returns the full destination path.
func StoreLocal(storageDir, jobName, runID, archivePath string) (string, int64, error) {
	if storageDir == "" {
		return "", 0, fmt.Errorf("LOCAL storage target has no path configured")
	}
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", 0, fmt.Errorf("create storage dir: %w", err)
	}
	base := strings.ToLower(safeName.ReplaceAllString(jobName, "_"))
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
	}
	if strings.HasSuffix(archivePath, ".enc") {
		ext += ".enc"
	}
	dest := filepath.Join(storageDir, fmt.Sprintf("%s_%s%s", base, runID, ext))

	if err := copyFile(archivePath, dest); err != nil {
		return "", 0, err
	}
	st, err := os.Stat(dest)
	if err != nil {
		return "", 0, err
	}
	return dest, st.Size(), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
