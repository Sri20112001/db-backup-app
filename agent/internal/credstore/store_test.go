package credstore

import (
	"os"
	"runtime"
	"testing"
)

func TestFileStoreRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	creds := Credentials{AgentID: "agent-123", AgentToken: "token-abc"}
	if err := s.Save(creds); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != creds {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
	// Owner-only permissions. On Unix this is a strict 0600; on Windows Go
	// cannot express ACLs via permission bits (files inherit directory
	// ACLs), so we only assert no group/other access where expressible.
	// At-rest secrecy on Windows comes from the DPAPI backend (opt-in until
	// runtime-validated) plus ProgramData's default ACLs.
	fi, err := os.Stat(s.Path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0077 != 0 {
		t.Errorf("credential file too permissive: %o", fi.Mode().Perm())
	}
	if err := s.Delete(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Load(); err == nil {
		t.Error("expected error after delete")
	}
	// Delete is idempotent.
	if err := s.Delete(); err != nil {
		t.Errorf("second delete: %v", err)
	}
}

func TestFileStoreRefusesIncomplete(t *testing.T) {
	s := NewFileStore(t.TempDir())
	for _, c := range []Credentials{{}, {AgentID: "x"}, {AgentToken: "y"}} {
		if err := s.Save(c); err == nil {
			t.Errorf("must refuse %+v", c)
		}
	}
}

func TestFileStoreLoadMissing(t *testing.T) {
	s := NewFileStore(t.TempDir())
	if _, err := s.Load(); err == nil {
		t.Error("expected error for missing file")
	}
}
