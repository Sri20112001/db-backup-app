//go:build windows

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// FreeDiskSpace reports bytes available to the caller on the volume holding
// dir. Used before S3 downloads so a huge restore fails fast with
// INSUFFICIENT_DISK_SPACE instead of filling the disk.
func FreeDiskSpace(dir string) (uint64, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return 0, err
	}
	var free uint64
	if err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(abs),
		&free, nil, nil,
	); err != nil {
		return 0, fmt.Errorf("disk space query: %w", err)
	}
	return free, nil
}
