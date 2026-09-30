package config

import (
	"bytes"
	"testing"
)

func TestResolveEncryptionKeyHex(t *testing.T) {
	// 64 hex chars must decode to the exact 32 bytes — not the ASCII text.
	raw := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	got := resolveEncryptionKey(raw)
	want := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("hex key decoded to %x, want %x", got, want)
	}
}

func TestResolveEncryptionKeyLegacy(t *testing.T) {
	// Pre-v1 deployments stored the first 32 raw bytes; that derivation
	// must keep working so old storage targets still decrypt.
	raw := "abcdefghijklmnopqrstuvwxyz0123456789-extra-ignored"
	got := resolveEncryptionKey(raw)
	if string(got) != "abcdefghijklmnopqrstuvwxyz012345" {
		t.Fatalf("legacy key derived to %q", got)
	}
}

func TestResolveEncryptionKeyTooShort(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for short key")
		}
	}()
	resolveEncryptionKey("too-short")
}

func TestOverrideSSLModeKV(t *testing.T) {
	got := overrideSSLMode("host=db user=u password=p dbname=d port=5432 sslmode=disable", "verify-full")
	if got != "host=db user=u password=p dbname=d port=5432 sslmode=verify-full" {
		t.Fatalf("got %q", got)
	}
}

func TestOverrideSSLModeKVAppend(t *testing.T) {
	got := overrideSSLMode("host=db user=u dbname=d", "require")
	if got != "host=db user=u dbname=d sslmode=require" {
		t.Fatalf("got %q", got)
	}
}

func TestOverrideSSLModeURL(t *testing.T) {
	got := overrideSSLMode("postgres://u:p@db:5432/d?sslmode=disable", "verify-full")
	if got != "postgres://u:p@db:5432/d?sslmode=verify-full" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEncryptionKeyBadHexFallsBack(t *testing.T) {
	// 64 chars that are NOT valid hex must not silently decode to garbage —
	// they take the legacy path (with warning) rather than failing.
	raw := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	got := resolveEncryptionKey(raw)
	if len(got) != 32 || string(got[:4]) != "zzzz" {
		t.Fatalf("non-hex 64-char key mishandled: %q", got)
	}
}
