package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/crypto"
)

// TOTP constants per RFC 6238.
const (
	TOTPDigits    = 6
	TOTPTimeStep  = 30 // seconds
	TOTPSecretLen = 20 // 160 bits
)

// TOTP handles Time-based One-Time Password operations.
type TOTP struct {
	skew int // Number of time steps to check on each side (default: 1)
}

// NewTOTP creates a new TOTP instance with default skew.
func NewTOTP() *TOTP {
	return &TOTP{skew: 1}
}

// SetSkew sets the number of time steps to check on each side for clock drift.
func (t *TOTP) SetSkew(skew int) {
	t.skew = skew
}

// GenerateSecret generates a new TOTP secret and otpauth URL.
func (t *TOTP) GenerateSecret(username string) (secret string, otpauthURL string, err error) {
	// Generate random 20-byte secret
	secretBytes, err := crypto.RandomBytes(TOTPSecretLen)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Encode as base32 (no padding)
	secret = base32.StdEncoding.EncodeToString(secretBytes)

	// Build otpauth URL
	// Format: otpauth://totp/VaultDrift:{username}?secret={secret}&issuer=VaultDrift
	otpauthURL = fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s",
		"VaultDrift",
		url.QueryEscape(username),
		secret,
		"VaultDrift",
	)

	return secret, otpauthURL, nil
}

// ValidateCode validates a TOTP code against a secret.
// Allows ±1 time step for clock drift.
func (t *TOTP) ValidateCode(secret, code string) bool {
	// Remove any spaces from code
	code = strings.ReplaceAll(code, " ", "")

	// Validate code format
	if len(code) != TOTPDigits {
		return false
	}

	// Decode secret
	secretBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	// Get current time step
	now := time.Now().Unix()
	currentStep := now / TOTPTimeStep

	// Check current and adjacent time steps for clock drift
	for i := -t.skew; i <= t.skew; i++ {
		expectedCode := generateTOTP(secretBytes, currentStep+int64(i))
		if subtleConstantTimeCompare(code, expectedCode) {
			return true
		}
	}

	return false
}

// generateTOTP generates a TOTP code for a given time step.
func generateTOTP(secret []byte, timeStep int64) string {
	// Encode time step as 8-byte big-endian
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(timeStep))

	// HMAC-SHA1
	h := hmac.New(sha1.New, secret)
	h.Write(counter)
	hash := h.Sum(nil)

	// Dynamic truncation (RFC 4226)
	offset := hash[len(hash)-1] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset : offset+4])
	code &= 0x7fffffff // Clear most significant bit
	code %= uint32(pow10(TOTPDigits))

	// Format as zero-padded string
	return fmt.Sprintf("%0*d", TOTPDigits, code)
}

// GenerateCode generates a TOTP code for the current time (for testing).
func (t *TOTP) GenerateCode(secret string) (string, error) {
	secretBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	now := time.Now().Unix()
	timeStep := now / TOTPTimeStep
	return generateTOTP(secretBytes, timeStep), nil
}

// pow10 returns 10^n.
func pow10(n int) int {
	result := 1
	for range n {
		result *= 10
	}
	return result
}

// subtleConstantTimeCompare compares two strings in constant time.
func subtleConstantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range len(a) {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// BackupCode represents a single backup code.
type BackupCode struct {
	Code   string
	Used   bool
	UsedAt *time.Time
}

// GenerateBackupCodes generates 10 random backup codes.
func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, 10)

	for i := 0; i < 10; i++ {
		// Generate 8 random bytes (64 bits)
		bytes, err := crypto.RandomBytes(8)
		if err != nil {
			return nil, err
		}

		// Encode as hex, take first 16 chars, format as XXXX-XXXX-XXXX-XXXX
		hex := fmt.Sprintf("%x", bytes)
		code := fmt.Sprintf("%s-%s-%s-%s", hex[0:4], hex[4:8], hex[8:12], hex[12:16])
		codes[i] = code
	}

	return codes, nil
}

// ValidateBackupCode validates a backup code against a list.
// Returns the index of the matching code or -1 if not found.
func ValidateBackupCode(code string, codes []BackupCode) int {
	code = strings.ReplaceAll(strings.ToUpper(code), " ", "")
	code = strings.ReplaceAll(code, "-", "")

	for i, bc := range codes {
		if bc.Used {
			continue
		}

		bcCode := strings.ReplaceAll(strings.ToUpper(bc.Code), "-", "")
		if subtleConstantTimeCompare(code, bcCode) {
			return i
		}
	}

	return -1
}
