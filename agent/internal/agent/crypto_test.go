package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "plain.bin")
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCryptoRoundTrip(t *testing.T) {
	// 3 MiB exercises multi-chunk framing (1 MiB chunks).
	plain := bytes.Repeat([]byte("abcdefgh12345678"), 3<<17)
	src := writeTemp(t, plain)
	enc := filepath.Join(t.TempDir(), "enc.bin")
	dec := filepath.Join(t.TempDir(), "dec.bin")

	key, err := GenerateDataKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncryptFile(key, src, enc); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(mustRead(t, src), mustRead(t, enc)) {
		t.Fatal("ciphertext must differ from plaintext")
	}
	if err := DecryptFile(key, enc, dec); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, mustRead(t, dec)) {
		t.Fatal("decrypted output differs from input")
	}
}

func TestCryptoEmptyFile(t *testing.T) {
	src := writeTemp(t, []byte{})
	enc := filepath.Join(t.TempDir(), "enc.bin")
	dec := filepath.Join(t.TempDir(), "dec.bin")
	key, _ := GenerateDataKey()
	if _, err := EncryptFile(key, src, enc); err != nil {
		t.Fatal(err)
	}
	if err := DecryptFile(key, enc, dec); err != nil {
		t.Fatal(err)
	}
	if len(mustRead(t, dec)) != 0 {
		t.Fatal("expected empty output")
	}
}

func TestCryptoWrongKey(t *testing.T) {
	src := writeTemp(t, []byte("secret"))
	enc := filepath.Join(t.TempDir(), "enc.bin")
	k1, _ := GenerateDataKey()
	k2, _ := GenerateDataKey()
	if _, err := EncryptFile(k1, src, enc); err != nil {
		t.Fatal(err)
	}
	if err := DecryptFile(k2, enc, filepath.Join(t.TempDir(), "dec.bin")); err == nil {
		t.Fatal("expected authentication failure with wrong key")
	}
}

func TestCryptoTampered(t *testing.T) {
	src := writeTemp(t, bytes.Repeat([]byte("x"), 100))
	enc := filepath.Join(t.TempDir(), "enc.bin")
	key, _ := GenerateDataKey()
	if _, err := EncryptFile(key, src, enc); err != nil {
		t.Fatal(err)
	}
	raw := mustRead(t, enc)
	raw[len(raw)-1] ^= 0xff // flip last ciphertext byte
	if err := os.WriteFile(enc, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := DecryptFile(key, enc, filepath.Join(t.TempDir(), "dec.bin")); err == nil {
		t.Fatal("expected authentication failure on tampered file")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
