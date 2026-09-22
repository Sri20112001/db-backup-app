package agent

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// ProgressFunc receives cumulative byte counters during a backup.
type ProgressFunc func(bytesRead, bytesCompressed int64)

// BackupResult describes a finished local archive.
type BackupResult struct {
	ArchivePath     string
	BytesRead       int64
	BytesCompressed int64
	Checksum        string // hex sha256 of the archive file
}

func splitPatterns(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// matchAny reports whether rel (slash-separated, relative to source root)
// matches any of the patterns. A pattern matches the full relative path or
// any trailing portion (so "*.log" matches "sub/dir/app.log", and "docs"
// matches the "docs" subtree).
func matchAny(rel string, patterns []string) bool {
	base := path.Base(rel)
	for _, p := range patterns {
		p = filepath.ToSlash(p)
		if ok, _ := path.Match(p, rel); ok {
			return true
		}
		if ok, _ := path.Match(p, base); ok {
			return true
		}
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

func included(rel string, include, exclude []string) bool {
	if rel == "." {
		return true
	}
	if len(include) > 0 && !matchAny(rel, include) {
		return false
	}
	if matchAny(rel, exclude) {
		return false
	}
	return true
}

// BackupFilesystem archives sourcePath into a temp .tar(.gz) file.
// mode is NORMAL (plain tar) or COMPRESSED (tar+gzip).
func BackupFilesystem(sourcePath, mode string, include, exclude []string, progress ProgressFunc) (*BackupResult, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("source not accessible: %w", err)
	}

	compressed := strings.ToUpper(mode) == "COMPRESSED"
	ext := ".tar"
	if compressed {
		ext = ".tar.gz"
	}
	tmp, err := os.CreateTemp("", "backup-*"+ext)
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()

	hash := sha256.New()
	fileWriter := io.MultiWriter(tmp, hash)

	var tw *tar.Writer
	var gz *gzip.Writer
	if compressed {
		gz, _ = gzip.NewWriterLevel(fileWriter, gzip.BestCompression)
		tw = tar.NewWriter(gz)
	} else {
		tw = tar.NewWriter(fileWriter)
	}

	var bytesRead int64
	lastReport := time.Now()
	report := func() {
		if progress == nil {
			return
		}
		// bytesCompressed unknown until close; report bytesRead live.
		progress(bytesRead, 0)
	}

	singleFile := !info.IsDir()
	writeEntry := func(fullPath, rel string, fi os.FileInfo) error {
		// Preserve symlinks as links; skip sockets/pipes/devices.
		var linkTarget string
		if fi.Mode()&os.ModeSymlink != 0 {
			t, err := os.Readlink(fullPath)
			if err != nil {
				return err
			}
			linkTarget = t
		}
		hdr, err := tar.FileInfoHeader(fi, linkTarget)
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			f, err := os.Open(fullPath)
			if err != nil {
				return err
			}
			n, err := io.Copy(tw, f)
			f.Close()
			if err != nil {
				return err
			}
			bytesRead += n
			if time.Since(lastReport) > 2*time.Second {
				lastReport = time.Now()
				report()
			}
		}
		return nil
	}

	if singleFile {
		rel := filepath.Base(sourcePath)
		if err := writeEntry(sourcePath, rel, info); err != nil {
			tw.Close()
			if gz != nil {
				gz.Close()
			}
			tmp.Close()
			os.Remove(tmpPath)
			return nil, err
		}
	} else {
		root := filepath.Clean(sourcePath)
		err = filepath.Walk(root, func(fullPath string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, fullPath)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if !included(rel, include, exclude) {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if rel == "." {
				return nil
			}
			if !fi.Mode().IsRegular() && !fi.IsDir() && fi.Mode()&os.ModeSymlink == 0 {
				return nil // skip sockets, pipes, devices
			}
			return writeEntry(fullPath, rel, fi)
		})
		if err != nil {
			tw.Close()
			if gz != nil {
				gz.Close()
			}
			tmp.Close()
			os.Remove(tmpPath)
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return nil, err
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return nil, err
		}
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	st, err := os.Stat(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	report()
	return &BackupResult{
		ArchivePath:     tmpPath,
		BytesRead:       bytesRead,
		BytesCompressed: st.Size(),
		Checksum:        hex.EncodeToString(hash.Sum(nil)),
	}, nil
}
