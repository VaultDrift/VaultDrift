package util

import (
	"testing"
)

func TestHashBytes(t *testing.T) {
	data := []byte("hello world")
	hash := HashBytes(data)

	// SHA-256 hash of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("HashBytes() = %s, want %s", hash, expected)
	}
}

func TestHashString(t *testing.T) {
	data := "hello world"
	hash := HashString(data)

	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("HashString() = %s, want %s", hash, expected)
	}
}

func TestValidateHash(t *testing.T) {
	tests := []struct {
		name  string
		hash  string
		valid bool
	}{
		{
			name:  "valid hash",
			hash:  "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			valid: true,
		},
		{
			name:  "too short",
			hash:  "abc123",
			valid: false,
		},
		{
			name:  "invalid characters",
			hash:  "g94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateHash(tt.hash)
			if got != tt.valid {
				t.Errorf("ValidateHash() = %v, want %v", got, tt.valid)
			}
		})
	}
}
