package util

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// Reserved filenames on Windows
var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// reservedChars are characters not allowed in filenames
var reservedChars = `<>:"/\|?*`

// SanitizeFilename sanitizes a filename, removing or replacing invalid characters.
func SanitizeFilename(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Replace reserved characters
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if strings.ContainsRune(reservedChars, r) {
			b.WriteRune('_')
		} else if r < 32 {
			// Control characters
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	name = b.String()

	// Check for reserved names (Windows)
	upper := strings.ToUpper(name)
	if reservedNames[upper] {
		name = "_" + name
	}

	// Trim trailing dots and spaces (Windows)
	name = strings.TrimRight(name, ". ")

	// Ensure not empty
	if name == "" {
		name = "unnamed"
	}

	// Limit length
	if len(name) > 255 {
		name = name[:255]
	}

	return name
}

// IsValidFilename checks if a filename is valid (not sanitized).
func IsValidFilename(name string) bool {
	return name == SanitizeFilename(name)
}

// NormalizePath normalizes a path, cleaning it and ensuring it starts with /.
func NormalizePath(p string) string {
	// Clean the path
	p = path.Clean(p)

	// Ensure leading /
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// Ensure no trailing / (except for root)
	if p != "/" && strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}

	return p
}

// JoinPath joins path components.
func JoinPath(components ...string) string {
	return NormalizePath(path.Join(components...))
}

// SplitPath splits a path into its components.
func SplitPath(p string) []string {
	p = NormalizePath(p)
	if p == "/" {
		return []string{}
	}
	parts := strings.Split(p, "/")
	// Remove empty first element (from leading /)
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	return parts
}

// PathContains checks if path a contains path b (b is a subdirectory of a).
func PathContains(parent, child string) bool {
	parent = NormalizePath(parent)
	child = NormalizePath(child)

	if parent == child {
		return true
	}

	if !strings.HasSuffix(parent, "/") {
		parent += "/"
	}

	return strings.HasPrefix(child+"/", parent)
}

// ValidatePath checks if a path is safe (no traversal attacks).
func ValidatePath(p string) error {
	// Check for null bytes
	if strings.ContainsRune(p, '\x00') {
		return fmt.Errorf("path contains null byte")
	}

	// Normalize and check
	normalized := NormalizePath(p)

	// Check components
	parts := SplitPath(normalized)
	for _, part := range parts {
		if part == "." || part == ".." {
			return fmt.Errorf("path contains relative component: %s", part)
		}
		if part == "" {
			return fmt.Errorf("path contains empty component")
		}
		if len(part) > 255 {
			return fmt.Errorf("path component too long: %s", part)
		}
	}

	return nil
}

// GetFilename returns the filename (last component) from a path.
func GetFilename(p string) string {
	return path.Base(p)
}

// GetDir returns the directory (all but last component) from a path.
func GetDir(p string) string {
	return path.Dir(NormalizePath(p))
}

// GetExt returns the file extension (lowercase, with dot).
func GetExt(p string) string {
	ext := path.Ext(p)
	return strings.ToLower(ext)
}

// StripExt returns the filename without extension.
func StripExt(p string) string {
	ext := path.Ext(p)
	if ext == "" {
		return p
	}
	return p[:len(p)-len(ext)]
}

// SanitizePath sanitizes a full path (both directory and filename components).
func SanitizePath(p string) string {
	parts := SplitPath(p)
	for i, part := range parts {
		parts[i] = SanitizeFilename(part)
	}
	return JoinPath(parts...)
}

// IsPrintable checks if a string contains only printable characters.
func IsPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
