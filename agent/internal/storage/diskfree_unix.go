//go:build !windows

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// FreeDiskSpace reports bytes available to unprivileged callers on the
// filesystem holding dir. Used before S3 downloads so a huge restore fails
// fast with INSUFFICIENT_DISK_SPACE instead of filling the disk.
func FreeDiskSpace(dir string) (uint64, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return 0, err
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(abs, &stat); err != nil {
		return 0, fmt.Errorf("statfs: %w", err)
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
