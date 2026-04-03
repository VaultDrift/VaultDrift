package server

import (
	"strings"
	"unicode"
)

// sanitizeFilename sanitizes a filename for safe use in HTTP headers.
// It removes control characters, newlines, and quotes that could be used
// for HTTP header injection attacks.
func sanitizeFilename(name string) string {
	// Replace dangerous characters
	var sb strings.Builder
	for _, r := range name {
		switch {
		// Reject control characters, newlines, tabs
		case r < 32:
			sb.WriteRune('_')
		// Reject quotes that could break out of the header value
		case r == '"':
			sb.WriteRune('\'')
		// Reject backslash
		case r == '\\':
			sb.WriteRune('_')
		// Replace other potentially dangerous chars
		case unicode.IsControl(r):
			sb.WriteRune('_')
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
