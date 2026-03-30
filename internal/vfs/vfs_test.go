package vfs

import (
	"testing"
)

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"valid.txt", true},
		{"folder-name", true},
		{"file with spaces.txt", true},
		{"", false},
		{".", false},
		{"..", false},
		{"file/name.txt", false},
		{"file\x00null.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidName(tt.name)
			if result != tt.expected {
				t.Errorf("isValidName(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	prefix := "file"
	id := generateID(prefix)

	if id == "" {
		t.Error("Generated ID should not be empty")
	}

	if len(id) <= len(prefix) {
		t.Error("Generated ID should be longer than prefix")
	}

	// Should start with prefix
	if id[:len(prefix)] != prefix {
		t.Errorf("ID should start with %s, got %s", prefix, id)
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"folder"}, "folder"},
		{[]string{"folder", "subfolder"}, "folder/subfolder"},
		{[]string{"a", "b", "c"}, "a/b/c"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := joinPath(tt.parts)
			if result != tt.expected {
				t.Errorf("joinPath(%v) = %q, want %q", tt.parts, result, tt.expected)
			}
		})
	}
}
