package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// minimalPE builds a tiny but header-valid PE image with the given machine.
func minimalPE(machine uint16) []byte {
	raw := make([]byte, 512)
	raw[0], raw[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(raw[0x3C:], 64)
	raw[64], raw[65], raw[66], raw[67] = 'P', 'E', 0, 0
	binary.LittleEndian.PutUint16(raw[68:], machine)
	return raw
}

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "artifact.exe")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestVerifyFallbackPE(t *testing.T) {
	// debug/pe cannot parse this (no sections), but the manual fallback
	// must still gate architecture correctly.
	ok := writeTemp(t, minimalPE(0x8664))
	if err := verify(ok, "amd64", 10); err != nil {
		t.Errorf("valid amd64 header rejected: %v", err)
	}
	if err := verify(ok, "386", 10); err == nil {
		t.Error("wrong arch must be rejected")
	}
	if err := verify(ok, "amd64", 1<<20); err == nil {
		t.Error("undersized file must be rejected")
	}
	if err := verify(writeTemp(t, []byte("not a binary at all............")), "amd64", 1); err == nil {
		t.Error("garbage must be rejected")
	}
}

func TestPeMachineFallback(t *testing.T) {
	m, err := peMachineFallback(minimalPE(0x8664))
	if err != nil || m != 0x8664 {
		t.Errorf("got 0x%04X, %v", m, err)
	}
	if _, err := peMachineFallback([]byte("short")); err == nil {
		t.Error("truncated input must fail")
	}
}
