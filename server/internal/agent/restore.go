package agent

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RestoreLocal extracts a .tar(.gz) archive into destDir.
func RestoreLocal(archivePath, destDir string, progress ProgressFunc) (int64, error) {
	if destDir == "" {
		return 0, fmt.Errorf("destination_path is required for FILESYSTEM restore")
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("create destination dir: %w", err)
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	var reader io.Reader = f
	if strings.HasSuffix(archivePath, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return 0, fmt.Errorf("open gzip: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	var written int64
	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return written, fmt.Errorf("read archive: %w", err)
		}
		// Zip-slip protection: keep everything inside destDir.
		target := filepath.Join(destDir, filepath.FromSlash(hdr.Name))
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(destDir) {
			return written, fmt.Errorf("archive entry escapes destination: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return written, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return written, err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return written, err
			}
			n, err := io.Copy(out, tr)
			out.Close()
			if err != nil {
				return written, err
			}
			written += n
			if progress != nil {
				progress(written, written)
			}
		case tar.TypeSymlink:
			os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return written, err
			}
		default:
			// skip devices, pipes, etc.
		}
	}
	return written, nil
}
