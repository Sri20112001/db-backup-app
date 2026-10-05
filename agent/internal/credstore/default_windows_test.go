//go:build windows

package credstore

import (
	"path/filepath"
	"strings"
	"testing"
)

// On Windows the default store is DPAPI/LocalMachine (service-compatible)
// with transparent adoption of pre-DPAPI plaintext files.
func TestDefaultStoreIsDPAPI(t *testing.T) {
	def, ok := NewDefaultStore(t.TempDir()).(*DPAPIStore)
	if !ok {
		t.Fatalf("windows default must be DPAPI, got %T", NewDefaultStore(t.TempDir()))
	}
	if def.scope != ScopeLocalMachine {
		t.Errorf("windows default scope must be LocalMachine, got %d", def.scope)
	}
	if !strings.HasSuffix(def.file.Path, "agent-credential.json") {
		t.Errorf("unexpected path %q", def.file.Path)
	}
}

func TestDPAPIStoreAdoptsPlaintext(t *testing.T) {
	dir := t.TempDir()
	// Simulate a pre-DPAPI install: plaintext credential file in place.
	if err := NewFileStore(dir).Save(Credentials{AgentID: "a", AgentToken: "tok"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := NewDPAPIStoreWithScope(dir, ScopeLocalMachine).Load()
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if got.AgentID != "a" || got.AgentToken != "tok" {
		t.Errorf("adopted mismatch: %+v", got)
	}
	// Self-healing: the file must now be an opaque DPAPI blob, not plaintext.
	raw, err := readFile(filepath.Join(dir, "agent-credential.json"))
	if err != nil {
		t.Fatalf("reread: %v", err)
	}
	if strings.Contains(string(raw), "tok") {
		t.Error("adopted credential file must be re-protected at rest")
	}
	// …and still loadable afterwards.
	if _, err := NewDPAPIStoreWithScope(dir, ScopeLocalMachine).Load(); err != nil {
		t.Errorf("reload after reprotect: %v", err)
	}
}
