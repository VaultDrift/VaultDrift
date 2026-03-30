package util

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// Base64URLEncode encodes data to URL-safe base64 (no padding).
func Base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// Base64URLDecode decodes URL-safe base64 data.
func Base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	if len(s)%4 != 0 {
		s += strings.Repeat("=", 4-len(s)%4)
	}
	return base64.URLEncoding.DecodeString(s)
}

// Base64StdEncode encodes data to standard base64.
func Base64StdEncode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64StdDecode decodes standard base64 data.
func Base64StdDecode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// HexEncode encodes data to hex string.
func HexEncode(data []byte) string {
	return hex.EncodeToString(data)
}

// HexDecode decodes a hex string to bytes.
func HexDecode(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// SafeToken generates a URL-safe random token from random bytes.
func SafeToken(randomBytes []byte) string {
	return Base64URLEncode(randomBytes)
}

// EscapeJSON escapes a string for safe inclusion in JSON.
func EscapeJSON(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
