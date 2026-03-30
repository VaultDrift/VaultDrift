package util

import (
	"fmt"
	"time"
)

// FormatRFC3339 formats a time as RFC 3339 (for JSON APIs).
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseRFC3339 parses an RFC 3339 formatted time string.
func ParseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// FormatDateTime formats a time as a human-readable date and time.
func FormatDateTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

// FormatDate formats a time as a date only.
func FormatDate(t time.Time) string {
	return t.Local().Format("2006-01-02")
}

// FormatRelative formats a time as a relative string (e.g., "2 hours ago").
func FormatRelative(t time.Time) string {
	diff := time.Since(t)
	if diff < 0 {
		diff = -diff
		return formatRelativeFuture(diff)
	}
	return formatRelativePast(diff)
}

func formatRelativePast(d time.Duration) string {
	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		s := int(d.Seconds())
		if s == 1 {
			return "1 second ago"
		}
		return fmt.Sprintf("%d seconds ago", s)
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

func formatRelativeFuture(d time.Duration) string {
	switch {
	case d < time.Second:
		return "in a moment"
	case d < time.Minute:
		s := int(d.Seconds())
		if s == 1 {
			return "in 1 second"
		}
		return fmt.Sprintf("in %d seconds", s)
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "in 1 hour"
		}
		return fmt.Sprintf("in %d hours", h)
	default:
		return FormatDateTime(time.Now().Add(d))
	}
}

// FormatDuration formats a duration as a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "-" + FormatDuration(-d)
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		if hours == 0 && minutes == 0 && seconds == 0 {
			if days == 1 {
				return "1 day"
			}
			return fmt.Sprintf("%d days", days)
		}
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}

	if hours > 0 {
		if minutes == 0 && seconds == 0 {
			if hours == 1 {
				return "1 hour"
			}
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	if minutes > 0 {
		if seconds == 0 {
			if minutes == 1 {
				return "1 minute"
			}
			return fmt.Sprintf("%d minutes", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	if seconds == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", seconds)
}

// ParseDuration parses a duration string supporting days (d).
func ParseDuration(s string) (time.Duration, error) {
	// Handle days suffix specially
	var days int
	if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Fall back to standard parsing
	return time.ParseDuration(s)
}

// StartOfDay returns the start of the day for a given time.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day for a given time.
func EndOfDay(t time.Time) time.Time {
	return StartOfDay(t).Add(24*time.Hour - time.Nanosecond)
}

// TruncateToDay truncates a time to the start of its day.
func TruncateToDay(t time.Time) time.Time {
	return StartOfDay(t)
}
