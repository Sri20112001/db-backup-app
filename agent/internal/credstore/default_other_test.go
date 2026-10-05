//go:build !windows

package credstore

import (
	"strings"
	"testing"
)

// Off Windows the default store is the 0600 file store.
func TestDefaultStoreIsFile(t *testing.T) {
	def, ok := NewDefaultStore(t.TempDir()).(*FileStore)
	if !ok {
		t.Fatalf("non-windows default must be FileStore, got %T", NewDefaultStore(t.TempDir()))
	}
	if !strings.HasSuffix(def.Path, "agent-credential.json") {
		t.Errorf("unexpected path %q", def.Path)
	}
}
