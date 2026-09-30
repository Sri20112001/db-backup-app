package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

type tarEntry struct {
	name     string
	body     string
	typeflag byte
	linkname string
}

func writeTar(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0600,
			Size:     int64(len(e.body)),
			Typeflag: e.typeflag,
			Linkname: e.linkname,
		}
		if e.typeflag == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typeflag == tar.TypeReg || e.typeflag == 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "evil.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestRestoreRejectsSymlinkEscape reproduces the classic attack: an archive
// plants `link -> <outside>` and then a regular `link/passwd` entry. The
// string-prefix check sees `dest/link/passwd` as inside dest, but the
// kernel would resolve it outside. Restore must fail closed.
func TestRestoreRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "passwd")
	if err := os.WriteFile(sentinel, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "dest")

	archive := writeTar(t, []tarEntry{
		{name: "link", typeflag: tar.TypeSymlink, linkname: outside},
		{name: "link/passwd", body: "pwned"},
	})
	if _, err := RestoreLocal(archive, dest, nil); err == nil {
		t.Fatal("expected error restoring symlink escape archive")
	}
	if got := mustRead(t, sentinel); string(got) != "original" {
		t.Fatalf("outside file modified: %q", got)
	}
	if _, err := os.Lstat(filepath.Join(dest, "link")); !os.IsNotExist(err) {
		t.Fatal("symlink must not be created in destination")
	}
}

func TestRestoreRejectsHardlink(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	archive := writeTar(t, []tarEntry{
		{name: "real.txt", body: "data"},
		{name: "hard.txt", typeflag: tar.TypeLink, linkname: "real.txt"},
	})
	if _, err := RestoreLocal(archive, dest, nil); err == nil {
		t.Fatal("expected error restoring hardlink entry")
	}
}

func TestRestoreRejectsZipSlip(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	archive := writeTar(t, []tarEntry{
		{name: "../../escape.txt", body: "pwned"},
	})
	if _, err := RestoreLocal(archive, dest, nil); err == nil {
		t.Fatal("expected error restoring zip-slip entry")
	}
	if _, err := os.Lstat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Fatal("zip-slip file escaped destination")
	}
}

func TestRestoreRefusesSymlinkOverwrite(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing symlink at the exact target path (e.g. planted by an
	// older restore that still created links): writing through it must fail.
	if err := os.Symlink(root, filepath.Join(dest, "planted")); err != nil {
		t.Fatal(err)
	}
	archive := writeTar(t, []tarEntry{
		{name: "planted", body: "pwned"},
	})
	if _, err := RestoreLocal(archive, dest, nil); err == nil {
		t.Fatal("expected error overwriting a symlink")
	}
}

func TestRestoreHappyPath(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	archive := writeTar(t, []tarEntry{
		{name: "a.txt", body: "hello"},
		{name: "sub/b.txt", body: "world"},
	})
	n, err := RestoreLocal(archive, dest, nil)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if n != int64(len("hello")+len("world")) {
		t.Fatalf("written = %d, want %d", n, len("hello")+len("world"))
	}
	if got := mustRead(t, filepath.Join(dest, "sub", "b.txt")); string(got) != "world" {
		t.Fatalf("restored content = %q", got)
	}
}
