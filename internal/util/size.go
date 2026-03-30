package util

import (
	"fmt"
)

// Byte sizes
const (
	Byte = 1
	KB   = 1024 * Byte
	MB   = 1024 * KB
	GB   = 1024 * MB
	TB   = 1024 * GB
	PB   = 1024 * TB
)

// FormatSize formats a byte size as a human-readable string.
// Uses binary units (KiB, MiB, etc.) with 2 decimal places for sizes >= 1KB.
func FormatSize(bytes int64) string {
	if bytes < 0 {
		return "-" + FormatSize(-bytes)
	}

	switch {
	case bytes >= PB:
		return fmt.Sprintf("%.2f PB", float64(bytes)/PB)
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSizeDecimal formats a byte size using decimal units (KB = 1000 bytes).
func FormatSizeDecimal(bytes int64) string {
	if bytes < 0 {
		return "-" + FormatSizeDecimal(-bytes)
	}

	const (
		kb = 1000
		mb = 1000 * kb
		gb = 1000 * mb
		tb = 1000 * gb
		pb = 1000 * tb
	)

	switch {
	case bytes >= pb:
		return fmt.Sprintf("%.2f PB", float64(bytes)/pb)
	case bytes >= tb:
		return fmt.Sprintf("%.2f TB", float64(bytes)/tb)
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/gb)
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/mb)
	case bytes >= kb:
		return fmt.Sprintf("%.2f kB", float64(bytes)/kb)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ParseSize parses a human-readable size string to bytes.
// Supports units: B, KB, MB, GB, TB, PB (binary, 1024-based) and KiB, MiB, etc.
func ParseSize(s string) (int64, error) {
	var value float64
	var unit string

	_, err := fmt.Sscanf(s, "%f%s", &value, &unit)
	if err != nil {
		// Try without unit (bytes)
		_, err = fmt.Sscanf(s, "%f", &value)
		if err != nil {
			return 0, fmt.Errorf("invalid size format: %s", s)
		}
		return int64(value), nil
	}

	unit = normalizeUnit(unit)

	switch unit {
	case "b", "":
		return int64(value), nil
	case "kb", "kib":
		return int64(value * KB), nil
	case "mb", "mib":
		return int64(value * MB), nil
	case "gb", "gib":
		return int64(value * GB), nil
	case "tb", "tib":
		return int64(value * TB), nil
	case "pb", "pib":
		return int64(value * PB), nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
}

func normalizeUnit(unit string) string {
	// Convert to lowercase and remove spaces
	result := ""
	for _, r := range unit {
		if r >= 'A' && r <= 'Z' {
			result += string(r - 'A' + 'a')
		} else if r != ' ' {
			result += string(r)
		}
	}
	return result
}

// Percentage calculates a percentage, capped at 100.
func Percentage(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	p := float64(used) * 100.0 / float64(total)
	if p > 100 {
		return 100
	}
	return p
}

// FormatPercentage formats a percentage with 1 decimal place.
func FormatPercentage(p float64) string {
	return fmt.Sprintf("%.1f%%", p)
}
