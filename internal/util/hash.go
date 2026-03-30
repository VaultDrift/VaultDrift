// Package util provides shared utility functions for VaultDrift.
package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

// HashBytes computes the SHA-256 hash of data and returns it as a hex string.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HashString computes the SHA-256 hash of a string and returns it as a hex string.
func HashString(s string) string {
	return HashBytes([]byte(s))
}

// HashReader computes the SHA-256 hash of a reader's contents.
func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to hash reader: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashWriter wraps a writer and computes a hash of written data.
type HashWriter struct {
	w    io.Writer
	hash hash.Hash
}

// NewHashWriter creates a new HashWriter that computes SHA-256.
func NewHashWriter(w io.Writer) *HashWriter {
	return &HashWriter{
		w:    w,
		hash: sha256.New(),
	}
}

// Write writes data to the underlying writer and updates the hash.
func (hw *HashWriter) Write(p []byte) (n int, err error) {
	n, err = hw.w.Write(p)
	if n > 0 {
		hw.hash.Write(p[:n])
	}
	return n, err
}

// Sum returns the hex-encoded hash of all written data.
func (hw *HashWriter) Sum() string {
	return hex.EncodeToString(hw.hash.Sum(nil))
}

// HashEqual compares two hex-encoded hashes for equality.
// Returns true if both hashes are valid hex strings and equal.
func HashEqual(a, b string) bool {
	if len(a) != 64 || len(b) != 64 {
		return false
	}
	// Constant-time comparison to prevent timing attacks
	var v byte
	for i := 0; i < 64; i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// ValidateHash checks if a string is a valid SHA-256 hex hash.
func ValidateHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}
