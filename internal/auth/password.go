// Package auth provides authentication and authorization functionality.
package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vaultdrift/vaultdrift/internal/crypto"
)

// Password hashing constants.
const (
	// Argon2id-style parameters (simulated with PBKDF2 + memory-hard mixing)
	DefaultMemoryKB   = 65536 // 64MB
	DefaultIterations = 3
)

// ErrInvalidHash is returned when the hash format is invalid.
var ErrInvalidHash = errors.New("invalid password hash format")

// HashPassword hashes a password using PBKDF2 with memory-hard mixing.
// Returns a PHC-formatted string: $argon2id$v=19$m=65536,t=3,p=1$<salt>$<hash>
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using memory-hard function (Argon2id style)
	hash := crypto.DeriveKeyArgon2idStyle(password, salt, DefaultMemoryKB, DefaultIterations)

	// Encode as PHC string
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	// PHC format: $argon2id$v=19$m=65536,t=3,p=1$<salt>$<hash>
	phc := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=1$%s$%s",
		DefaultMemoryKB, DefaultIterations, encodedSalt, encodedHash)

	return phc, nil
}

// VerifyPassword verifies a password against a PHC-formatted hash.
func VerifyPassword(password, hash string) (bool, error) {
	// Parse PHC string
	salt, storedHash, memoryKB, iterations, err := parsePHC(hash)
	if err != nil {
		return false, err
	}

	// Derive key with same parameters
	derivedHash := crypto.DeriveKeyArgon2idStyle(password, salt, memoryKB, iterations)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(derivedHash, storedHash) != 1 {
		return false, nil
	}

	return true, nil
}

// parsePHC parses a PHC-formatted password hash.
// Format: $argon2id$v=19$m=65536,t=3,p=1$<salt>$<hash>
func parsePHC(phc string) (salt, hash []byte, memoryKB, iterations int, err error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	// parts[0] is empty (leading $)
	// parts[1] should be "argon2id"
	// parts[2] should be version (v=19)
	// parts[3] should be parameters (m=65536,t=3,p=1)
	// parts[4] is salt
	// parts[5] is hash

	if parts[1] != "argon2id" {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	// Parse parameters
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	// Parse memory (m=65536)
	memParts := strings.Split(params[0], "=")
	if len(memParts) != 2 || memParts[0] != "m" {
		return nil, nil, 0, 0, ErrInvalidHash
	}
	memoryKB, err = strconv.Atoi(memParts[1])
	if err != nil {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	// Parse time/iterations (t=3)
	timeParts := strings.Split(params[1], "=")
	if len(timeParts) != 2 || timeParts[0] != "t" {
		return nil, nil, 0, 0, ErrInvalidHash
	}
	iterations, err = strconv.Atoi(timeParts[1])
	if err != nil {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	// Decode hash
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, 0, 0, ErrInvalidHash
	}

	return salt, hash, memoryKB, iterations, nil
}
