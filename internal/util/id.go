package util

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"
)

// UUID v7 implementation based on draft-ietf-uuidrev-rfc4122bis.
// UUID v7 provides sortable, time-ordered UUIDs based on Unix timestamps.

var (
	lastTimestamp int64
	sequence      uint32
)

// GenerateUUIDv7 generates a new UUID v7.
// Returns a 36-character string in standard UUID format.
func GenerateUUIDv7() (string, error) {
	var uuid [16]byte

	// Get current timestamp in milliseconds
	timestamp := time.Now().UnixMilli()

	// Handle sequence for same-millisecond generation
	seq := uint32(0)
	for {
		last := atomic.LoadInt64(&lastTimestamp)
		if timestamp > last {
			if atomic.CompareAndSwapInt64(&lastTimestamp, last, timestamp) {
				atomic.StoreUint32(&sequence, 0)
				seq = 0
				break
			}
		} else if timestamp == last {
			seq = atomic.AddUint32(&sequence, 1) & 0x0FFF
			break
		} else {
			// Clock moved backward, use last timestamp
			timestamp = last
			seq = atomic.AddUint32(&sequence, 1) & 0x0FFF
			break
		}
	}

	// Encode timestamp (48 bits) in big-endian manually
	// #nosec G115 - Intentional byte extraction from 48-bit timestamp
	uuid[0] = byte(timestamp >> 40) // #nosec G115
	uuid[1] = byte(timestamp >> 32) // #nosec G115
	uuid[2] = byte(timestamp >> 24) // #nosec G115
	uuid[3] = byte(timestamp >> 16) // #nosec G115
	uuid[4] = byte(timestamp >> 8)  // #nosec G115
	uuid[5] = byte(timestamp)       // #nosec G115

	// Set version (4 bits) to 0b0111 (7)
	uuid[6] = (uuid[6] & 0x0F) | 0x70

	// Set variant (2 bits) to 0b10 (RFC 4122 variant)
	uuid[8] = (uuid[8] & 0x3F) | 0x80

	// Add sequence and random data
	// Use crypto/rand for remaining bytes
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode sequence in upper 12 bits of bytes 6-7
	uuid[6] |= byte(seq >> 8)
	uuid[7] = byte(seq & 0xFF)

	// Fill remaining bytes with random data
	copy(uuid[8:], randomBytes)

	return formatUUID(uuid), nil
}

// GenerateUUID returns a new UUID v7 string or panics on error.
// Use this when you need a UUID and can't handle errors.
func GenerateUUID() string {
	uuid, err := GenerateUUIDv7()
	if err != nil {
		panic(err)
	}
	return uuid
}

// formatUUID formats a 16-byte UUID as a standard string.
func formatUUID(uuid [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(uuid[0:4]),
		binary.BigEndian.Uint16(uuid[4:6]),
		binary.BigEndian.Uint16(uuid[6:8]),
		binary.BigEndian.Uint16(uuid[8:10]),
		uuid[10:],
	)
}

// ValidateUUID checks if a string is a valid UUID format.
func ValidateUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Check dashes at correct positions
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	// Check hex characters
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// RandomID generates a random ID of the specified length using alphanumeric characters.
func RandomID(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const alphabetLen = 62

	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := 0; i < length; i++ {
		result[i] = alphabet[int(randomBytes[i])%alphabetLen]
	}

	return string(result), nil
}

// RandomIDOrPanic generates a random ID or panics on error.
func RandomIDOrPanic(length int) string {
	id, err := RandomID(length)
	if err != nil {
		panic(err)
	}
	return id
}

// RandomToken generates a URL-safe random token of the specified length.
func RandomToken(length int) (string, error) {
	// URL-safe base64 alphabet without padding
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	const alphabetLen = 64

	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := 0; i < length; i++ {
		result[i] = alphabet[int(randomBytes[i])%alphabetLen]
	}

	return string(result), nil
}

// RandomTokenOrPanic generates a random token or panics on error.
func RandomTokenOrPanic(length int) string {
	token, err := RandomToken(length)
	if err != nil {
		panic(err)
	}
	return token
}

// RandomBytes generates cryptographically secure random bytes.
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// MustRandomBytes generates random bytes or panics on error.
func MustRandomBytes(n int) []byte {
	b, err := RandomBytes(n)
	if err != nil {
		panic(err)
	}
	return b
}

// SecureRandomInt returns a cryptographically secure random integer in [0, max).
func SecureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}
	if max <= 256 {
		b, err := RandomBytes(1)
		if err != nil {
			return 0, err
		}
		return int(b[0]) % max, nil
	}
	b, err := RandomBytes(4)
	if err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint32(b)) % max, nil
}

// SanitizeFileID validates and sanitizes a file ID for safe use in file paths.
// It returns the fileID if valid, or an error if the ID contains path traversal characters.
func SanitizeFileID(fileID string) (string, error) {
	// Check for path traversal attempts
	if fileID == "" || fileID == "." || fileID == ".." {
		return "", fmt.Errorf("invalid file ID")
	}
	// Check for path separators or parent directory references
	for _, c := range fileID {
		if c == '/' || c == '\\' || c == 0 {
			return "", fmt.Errorf("invalid file ID: contains path separator")
		}
	}
	return fileID, nil
}
