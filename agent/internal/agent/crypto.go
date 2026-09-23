package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const (
	// encMagic identifies our encrypted container format.
	encMagic = "VG01"
	// encChunkSize balances memory use against per-chunk GCM overhead.
	encChunkSize = 1 << 20 // 1 MiB
)

// GenerateDataKey creates a fresh 256-bit per-backup data key.
func GenerateDataKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate data key: %w", err)
	}
	return key, nil
}

func newAgentGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("data key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// nonceFor derives a unique 96-bit nonce per chunk: 4 random prefix bytes
// (per file) + 8-byte big-endian chunk counter. Uniqueness per (key, nonce)
// holds because keys are single-use per backup.
func nonceFor(prefix []byte, counter uint64) []byte {
	nonce := make([]byte, 12)
	copy(nonce, prefix)
	binary.BigEndian.PutUint64(nonce[4:], counter)
	return nonce
}

// EncryptFile encrypts srcPath with AES-256-GCM into dstPath and returns the
// hex SHA-256 of the ciphertext file. Format:
//
//	magic "VG01" | noncePrefix(4) | frames...
//	frame = nonce(12) | ctLen(4 BE) | ct
func EncryptFile(key []byte, srcPath, dstPath string) (string, error) {
	gcm, err := newAgentGCM(key)
	if err != nil {
		return "", err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open plaintext: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create ciphertext: %w", err)
	}
	defer dst.Close()

	hash := sha256.New()
	w := io.MultiWriter(dst, hash)

	if _, err := w.Write([]byte(encMagic)); err != nil {
		return "", err
	}
	prefix := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, prefix); err != nil {
		return "", err
	}
	if _, err := w.Write(prefix); err != nil {
		return "", err
	}

	buf := make([]byte, encChunkSize)
	var counter uint64
	for {
		n, rerr := io.ReadFull(src, buf)
		if n > 0 {
			nonce := nonceFor(prefix, counter)
			counter++
			ct := gcm.Seal(nil, nonce, buf[:n], nil)
			var hdr [16]byte
			copy(hdr[:12], nonce)
			binary.BigEndian.PutUint32(hdr[12:], uint32(len(ct)))
			if _, err := w.Write(hdr[:]); err != nil {
				return "", err
			}
			if _, err := w.Write(ct); err != nil {
				return "", err
			}
		}
		if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
			break
		}
		if rerr != nil {
			return "", fmt.Errorf("read plaintext: %w", rerr)
		}
	}
	if err := dst.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ChecksumFile returns the hex SHA-256 of a file's raw bytes.
func ChecksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// DecryptFile reverses EncryptFile. Any format error or authentication
// failure aborts with an error and removes the partial output.
func DecryptFile(key []byte, srcPath, dstPath string) error {
	gcm, err := newAgentGCM(key)
	if err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open ciphertext: %w", err)
	}
	defer src.Close()

	magic := make([]byte, len(encMagic))
	if _, err := io.ReadFull(src, magic); err != nil || string(magic) != encMagic {
		return fmt.Errorf("not a VaultGuard encrypted file")
	}
	if _, err := io.ReadFull(src, make([]byte, 4)); err != nil { // nonce prefix
		return fmt.Errorf("truncated header")
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create plaintext: %w", err)
	}
	ok := false
	defer func() {
		dst.Close()
		if !ok {
			os.Remove(dstPath)
		}
	}()

	for {
		var hdr [16]byte
		_, err := io.ReadFull(src, hdr[:])
		if err == io.EOF {
			break // clean end
		}
		if err == io.ErrUnexpectedEOF {
			return fmt.Errorf("truncated frame header")
		}
		if err != nil {
			return fmt.Errorf("read frame: %w", err)
		}
		n := binary.BigEndian.Uint32(hdr[12:])
		if n == 0 || n > encChunkSize+uint32(gcm.Overhead())+16 {
			return fmt.Errorf("corrupt frame length %d", n)
		}
		ct := make([]byte, n)
		if _, err := io.ReadFull(src, ct); err != nil {
			return fmt.Errorf("truncated frame body")
		}
		pt, err := gcm.Open(nil, hdr[:12], ct, nil)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		if _, err := dst.Write(pt); err != nil {
			return fmt.Errorf("write plaintext: %w", err)
		}
	}
	ok = true
	return dst.Close()
}
