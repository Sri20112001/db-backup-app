//go:build windows

package credstore

import (
	"strings"
	"testing"
)

// TestDPAPIRoundtrip exercises the real Windows DPAPI (CryptProtectData /
// CryptUnprotectData) in both scopes. Memory-only: nothing is written to
// disk, no admin rights needed. This is genuine Windows runtime validation
// of the crypto path (CI/Linux can only cross-compile it).
func TestDPAPIRoundtrip(t *testing.T) {
	plain := []byte(`{"agent_id":"agent-1","agent_token":"token-1"}`)
	for _, scope := range []Scope{ScopeCurrentUser, ScopeLocalMachine} {
		protected, err := dpapiProtect(plain, scope)
		if err != nil {
			t.Fatalf("scope %d protect: %v", scope, err)
		}
		if string(protected) == string(plain) {
			t.Fatalf("scope %d: blob not encrypted", scope)
		}
		back, err := dpapiUnprotect(protected)
		if err != nil {
			t.Fatalf("scope %d unprotect: %v", scope, err)
		}
		if string(back) != string(plain) {
			t.Fatalf("scope %d roundtrip mismatch", scope)
		}
	}
}

// TestDPAPIStoreRoundtrip covers the full store (protect → file → load)
// in a temp dir for both scopes.
func TestDPAPIStoreRoundtrip(t *testing.T) {
	creds := Credentials{AgentID: "agent-9", AgentToken: "token-9"}
	for _, scope := range []Scope{ScopeCurrentUser, ScopeLocalMachine} {
		dir := t.TempDir()
		s := NewDPAPIStoreWithScope(dir, scope)
		if err := s.Save(creds); err != nil {
			t.Fatalf("scope %d save: %v", scope, err)
		}
		// The file must be opaque: no credential material at rest.
		raw, err := readFile(s.file.Path)
		if err != nil {
			t.Fatalf("scope %d read: %v", scope, err)
		}
		if strings.Contains(string(raw), "token-9") {
			t.Fatalf("scope %d: credential readable at rest", scope)
		}
		got, err := s.Load()
		if err != nil {
			t.Fatalf("scope %d load: %v", scope, err)
		}
		if got != creds {
			t.Fatalf("scope %d mismatch: %+v", scope, got)
		}
		if err := s.Delete(); err != nil {
			t.Fatalf("scope %d delete: %v", scope, err)
		}
	}
}
