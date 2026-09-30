package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage persists objects as files under Root. Keys use forward
// slashes (see BuildKey) and map to nested directories. Location in the
// returned ObjectInfo is the absolute filesystem path — that is what the
// agent registers as the artifact's storage_path for LOCAL targets.
type LocalStorage struct {
	Root string
}

// NewLocalStorage validates the target directory eagerly so misconfigured
// jobs fail before the backup runs, not after it.
func NewLocalStorage(root string) (*LocalStorage, error) {
	if root == "" {
		return nil, fmt.Errorf("LOCAL storage target has no path configured")
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &LocalStorage{Root: root}, nil
}

func (s *LocalStorage) resolve(key string) (string, error) {
	// Legacy artifacts registered absolute filesystem paths instead of
	// keys: accept them iff they sit inside Root, so old recovery points
	// keep restoring after the key-layout migration.
	if filepath.IsAbs(key) {
		clean := filepath.Clean(key)
		if rel, err := filepath.Rel(s.Root, clean); err != nil || rel == ".." || isParentRef(rel) {
			return "", fmt.Errorf("storage path escapes root: %s", key)
		}
		return clean, nil
	}
	// filepath.Join collapses any ".." segments; verify the result is still
	// inside Root (zip-slip guard for crafted keys).
	dest := filepath.Join(s.Root, filepath.Clean(filepath.FromSlash(key)))
	if rel, err := filepath.Rel(s.Root, dest); err != nil || rel == ".." || isParentRef(rel) {
		return "", fmt.Errorf("storage key escapes root: %s", key)
	}
	return dest, nil
}

func isParentRef(rel string) bool {
	return len(rel) >= 3 && rel[:3] == ".."+string(os.PathSeparator)
}

// Put streams r to Root/key while hashing, so the recorded checksum covers
// exactly the bytes on disk.
func (s *LocalStorage) Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dest, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	out, err := os.Create(dest)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	n, err := io.Copy(out, io.TeeReader(r, hash))
	closeErr := out.Close()
	if err != nil {
		os.Remove(dest)
		return nil, err
	}
	if closeErr != nil {
		os.Remove(dest)
		return nil, closeErr
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	meta := map[string]string{
		MetaChecksumSHA256: sum,
		MetaVersion:        FormatVersion,
	}
	for k, v := range opts.Metadata {
		meta[k] = v
	}
	return &ObjectInfo{
		Key:            key,
		Size:           n,
		ChecksumSHA256: sum,
		Location:       dest,
		Metadata:       meta,
	}, nil
}

// Get opens the stored file for reading (restore / verification).
func (s *LocalStorage) Get(_ context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	dest, err := s.resolve(key)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(dest)
	if err != nil {
		return nil, nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, &ObjectInfo{Key: key, Size: st.Size(), Location: dest}, nil
}

// Delete removes the stored file. Missing files are a successful no-op so
// retention cleanup stays idempotent.
func (s *LocalStorage) Delete(_ context.Context, key string) error {
	dest, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Head stats the stored file.
func (s *LocalStorage) Head(_ context.Context, key string) (*ObjectInfo, error) {
	dest, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(dest)
	if err != nil {
		return nil, err
	}
	return &ObjectInfo{Key: key, Size: st.Size(), Location: dest}, nil
}
