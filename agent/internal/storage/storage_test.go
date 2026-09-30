package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactFilename(t *testing.T) {
	cases := []struct{ job, run, archive, want string }{
		{"Nightly DB", "r1", "/tmp/x.tar.gz", "nightly_db_r1.tar.gz"},
		{"Nightly DB", "r1", "/tmp/x.tar.gz.enc", "nightly_db_r1.tar.gz.enc"},
		{"ERP", "r2", "/tmp/dump.sql", "erp_r2.sql"},
		{"ERP", "r2", "/tmp/dump.sql.gz", "erp_r2.sql.gz"},
		{"MSSQL", "r3", "/tmp/a.bak", "mssql_r3.bak"},
		{"Mongo", "r4", "/tmp/a.archive.gz", "mongo_r4.archive.gz"},
		{"Weird/Name:*?", "r5", "/tmp/a.tar", "weird_name__r5.tar"},
		{"", "r6", "/tmp/a.tar", "backup_r6.tar"},
	}
	for _, c := range cases {
		if got := ArtifactFilename(c.job, c.run, c.archive); got != c.want {
			t.Errorf("ArtifactFilename(%q,%q,%q) = %q, want %q", c.job, c.run, c.archive, got, c.want)
		}
	}
}

func TestBuildKey(t *testing.T) {
	got := BuildKey("o1", "j1", "r1", "nightly_r1.tar.gz")
	want := "orgs/o1/jobs/j1/runs/r1/nightly_r1.tar.gz"
	if got != want {
		t.Fatalf("BuildKey = %q, want %q", got, want)
	}
	if strings.HasPrefix(got, "http") || strings.HasPrefix(got, "/") {
		t.Fatal("keys must be relative provider paths, never URLs")
	}
}

func TestLocalRoundTrip(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := NewLocalStorage(root)
	if err != nil {
		t.Fatal(err)
	}
	key := BuildKey("o", "j", "r", "a.tar.gz")
	body := "backup-bytes"
	info, err := s.Put(ctx, key, strings.NewReader(body), int64(len(body)), PutOptions{
		ContentType: ContentTypeForName("a.tar.gz"),
		Metadata:    map[string]string{MetaRunID: "r"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != int64(len(body)) || info.ChecksumSHA256 == "" {
		t.Fatalf("bad Put info: %+v", info)
	}
	if info.Metadata[MetaRunID] != "r" || info.Metadata[MetaVersion] != FormatVersion {
		t.Fatalf("metadata not propagated: %+v", info.Metadata)
	}
	wantPath := filepath.Join(root, filepath.FromSlash(key))
	if info.Location != wantPath {
		t.Fatalf("Location = %q, want %q", info.Location, wantPath)
	}

	head, err := s.Head(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if head.Size != int64(len(body)) {
		t.Fatalf("Head size = %d", head.Size)
	}

	rc, getInfo, err := s.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body || getInfo.Size != int64(len(body)) {
		t.Fatal("Get round-trip mismatch")
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Get(ctx, key); err == nil {
		t.Fatal("expected error reading deleted object")
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal("delete must be idempotent")
	}
}

func TestLocalRejectsEscape(t *testing.T) {
	ctx := context.Background()
	s, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Absolute path on another volume/directory (OS-correct absolute form).
	outside := filepath.Join(t.TempDir(), "evil")
	for _, key := range []string{"../escape", "a/../../escape", outside} {
		if _, err := s.Put(ctx, key, strings.NewReader("x"), 1, PutOptions{}); err == nil {
			t.Fatalf("Put accepted escaping key %q", key)
		}
		if _, _, err := s.Get(ctx, key); err == nil {
			t.Fatalf("Get accepted escaping key %q", key)
		}
	}
}

func TestLocalAcceptsLegacyAbsolutePath(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	legacy := filepath.Join(root, "job_run1.tar.gz")
	if err := os.WriteFile(legacy, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewLocalStorage(root)
	if err != nil {
		t.Fatal(err)
	}
	rc, _, err := s.Get(ctx, legacy)
	if err != nil {
		t.Fatalf("legacy absolute path rejected: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "old" {
		t.Fatal("legacy content mismatch")
	}
}

func TestFactoryRejectsUnknown(t *testing.T) {
	if _, err := ForJob("SMB", t.TempDir(), nil); err == nil {
		t.Fatal("expected error for unsupported storage type")
	}
	if _, err := ForJob("LOCAL", "", nil); err == nil {
		t.Fatal("expected error for empty local root")
	}
}
