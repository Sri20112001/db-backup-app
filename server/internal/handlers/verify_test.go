package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSha256File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	content := []byte("vaultguard-verify-me")
	if err := os.WriteFile(p, content, 0600); err != nil {
		t.Fatal(err)
	}
	sum, err := sha256File(p)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(content)
	if sum != hex.EncodeToString(want[:]) {
		t.Errorf("got %s, want %s", sum, hex.EncodeToString(want[:]))
	}
	if _, err := sha256File(filepath.Join(dir, "missing.bin")); err == nil {
		t.Error("expected error for missing file")
	}
}
