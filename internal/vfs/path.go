package vfs

import (
	"path/filepath"
	"strings"
)

// Separator is the path separator used in the virtual filesystem.
const Separator = "/"

// Normalize cleans and normalizes a virtual path.
func Normalize(path string) string {
	if path == "" {
		return Separator
	}

	// Ensure path starts with separator
	if !strings.HasPrefix(path, Separator) {
		path = Separator + path
	}

	// Clean the path (removes . and .., resolves duplicates)
	path = filepath.Clean(path)

	// filepath.Clean uses OS separator, convert back to /
	path = filepath.ToSlash(path)

	// Ensure it starts with /
	if !strings.HasPrefix(path, Separator) {
		path = Separator + path
	}

	return path
}

// Split splits a path into directory and base name.
func Split(path string) (dir, base string) {
	path = Normalize(path)

	// Remove trailing slash for processing
	path = strings.TrimSuffix(path, Separator)

	lastIdx := strings.LastIndex(path, Separator)
	if lastIdx == -1 {
		return Separator, path
	}

	if lastIdx == 0 {
		return Separator, path[1:]
	}

	return path[:lastIdx], path[lastIdx+1:]
}

// Join joins path elements into a single path.
func Join(elem ...string) string {
	if len(elem) == 0 {
		return Separator
	}

	// Normalize the first element
	result := Normalize(elem[0])

	// Join remaining elements
	for _, e := range rangeSlice(elem[1:]) {
		e = strings.Trim(e, Separator)
		if e != "" {
			result = result + Separator + e
		}
	}

	return Normalize(result)
}

// rangeSlice helper to handle range over slice
func rangeSlice(s []string) []string {
	return s
}

// Dir returns the directory portion of a path.
func Dir(path string) string {
	dir, _ := Split(path)
	return dir
}

// Base returns the last element of a path.
func Base(path string) string {
	_, base := Split(path)
	return base
}

// IsRoot checks if a path is the root path.
func IsRoot(path string) bool {
	return Normalize(path) == Separator
}

// Validate checks if a path is valid.
// Returns an error if the path contains invalid characters or patterns.
func Validate(path string) error {
	// Normalize first
	path = Normalize(path)

	// Check for empty path
	if path == "" {
		return ErrInvalidPath
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return ErrInvalidPath
	}

	// Split into components and validate each
	components := strings.Split(strings.Trim(path, Separator), Separator)
	for _, comp := range components {
		if comp == "" {
			continue // Skip empty from leading/trailing slashes
		}
		if !isValidComponent(comp) {
			return ErrInvalidPath
		}
	}

	return nil
}

// isValidComponent validates a single path component.
func isValidComponent(name string) bool {
	if name == "" {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	// Check for invalid characters
	if strings.ContainsAny(name, "\x00/") {
		return false
	}
	return true
}

// GetName extracts the name from a path (without parent directories).
func GetName(path string) string {
	return Base(path)
}

// GetParent returns the parent directory path.
func GetParent(path string) string {
	return Dir(path)
}

// Ext returns the file extension (including the dot).
func Ext(path string) string {
	base := Base(path)
	idx := strings.LastIndex(base, ".")
	if idx == -1 {
		return ""
	}
	return base[idx:]
}

// StripExt returns the path without the file extension.
func StripExt(path string) string {
	ext := Ext(path)
	if ext == "" {
		return path
	}
	return path[:len(path)-len(ext)]
}

// MatchPattern checks if a path matches a glob pattern.
func MatchPattern(pattern, path string) bool {
	// Simple glob matching: * matches any sequence, ? matches single char
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	pattern = "^" + pattern + "$"

	// Use regex for matching
	return matchRegex(pattern, path)
}

// matchRegex performs regex matching (simplified).
func matchRegex(pattern, s string) bool {
	// Simple implementation without regex package
	// Convert pattern to simple matching
	if pattern == "^.*$" {
		return true
	}

	// Handle leading ^ and trailing $
	if strings.HasPrefix(pattern, "^") {
		pattern = pattern[1:]
	}
	if strings.HasSuffix(pattern, "$") {
		pattern = pattern[:len(pattern)-1]
	}

	// Simple wildcard matching for common patterns
	if pattern == ".*" {
		return true
	}

	// More complex patterns - simplified matching
	parts := strings.Split(pattern, ".*")
	if len(parts) == 2 {
		// prefix.*suffix pattern
		prefix := strings.TrimSuffix(parts[0], "\\")
		suffix := strings.TrimPrefix(parts[1], "\\")
		return strings.HasPrefix(s, prefix) && strings.HasSuffix(s, suffix)
	}

	return pattern == s
}

// CommonPath returns the longest common path prefix.
func CommonPath(paths ...string) string {
	if len(paths) == 0 {
		return Separator
	}
	if len(paths) == 1 {
		return Dir(paths[0])
	}

	// Normalize all paths
	normalized := make([]string, len(paths))
	for i, p := range paths {
		normalized[i] = Normalize(p)
	}

	// Find common prefix
	first := normalized[0]
	for i := 1; i < len(normalized); i++ {
		first = commonPrefix(first, normalized[i])
		if first == Separator {
			break
		}
	}

	return first
}

// commonPrefix returns the common prefix of two paths.
func commonPrefix(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			// Back up to last separator
			for j := i - 1; j >= 0; j-- {
				if a[j] == '/' {
					return a[:j]
				}
			}
			return Separator
		}
	}

	// Return up to complete directory
	if len(a) > minLen && a[minLen] == '/' {
		return a[:minLen]
	}
	if len(b) > minLen && b[minLen] == '/' {
		return b[:minLen]
	}
	if minLen == 0 {
		return Separator
	}

	// Find last separator in common portion
	for i := minLen - 1; i >= 0; i-- {
		if a[i] == '/' {
			return a[:i]
		}
	}

	return Separator
}

// Rel returns a relative path from base to target.
func Rel(base, target string) (string, error) {
	base = Normalize(base)
	target = Normalize(target)

	// Target must be within base
	if !strings.HasPrefix(target, base) {
		return "", ErrInvalidPath
	}

	rel := strings.TrimPrefix(target, base)
	rel = strings.TrimPrefix(rel, Separator)

	if rel == "" {
		return ".", nil
	}

	return rel, nil
}
