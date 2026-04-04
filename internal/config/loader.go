package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Load loads configuration from a YAML file and applies environment variable overrides.
// If path is empty, only defaults and environment variables are used.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if provided
	if path != "" {
		data, err := os.ReadFile(path) // #nosec G304 - path is provided by admin/user
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, continue with defaults
		} else {
			if err := unmarshalYAML(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Apply environment variable overrides
	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	return cfg, nil
}

// loadFromEnv applies environment variable overrides to the config.
// Variables should be prefixed with VAULTDRIFT_ and use underscores for nesting.
// Example: VAULTDRIFT_SERVER_PORT=8080
func loadFromEnv(cfg *Config) error {
	prefix := "VAULTDRIFT_"

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Remove prefix and convert to lowercase
		key = strings.ToLower(strings.TrimPrefix(key, prefix))

		// Parse the key path (e.g., "server_port" -> ["server", "port"])
		path := strings.Split(key, "_")

		if err := setField(cfg, path, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	return nil
}

// setField sets a field in the config struct using a path of field names.
func setField(cfg *Config, path []string, value string) error {
	v := reflect.ValueOf(cfg).Elem()

	for i, name := range path {
		// Find the field by name (case-insensitive match)
		field := findField(v, name)
		if !field.IsValid() {
			return fmt.Errorf("unknown field: %s", strings.Join(path[:i+1], "."))
		}

		if i == len(path)-1 {
			// Set the final field value
			return setValue(field, value)
		}

		// Navigate deeper
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}
		if field.Kind() != reflect.Struct {
			return fmt.Errorf("cannot navigate into non-struct: %s", strings.Join(path[:i+1], "."))
		}
		v = field
	}

	return nil
}

// findField finds a field in a struct by name (case-insensitive).
func findField(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Match against yaml tag or field name
		yamlTag := field.Tag.Get("yaml")
		yamlName := strings.Split(yamlTag, ",")[0]

		if strings.EqualFold(field.Name, name) || yamlName == name {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

// setValue sets a reflect.Value from a string based on its type.
func setValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle duration fields specially
		if field.Type().String() == "time.Duration" {
			d, err := parseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
		} else {
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer: %s", value)
			}
			field.SetInt(n)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer: %s", value)
		}
		field.SetUint(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		field.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %s", value)
		}
		field.SetFloat(f)
	case reflect.Slice:
		elemType := field.Type().Elem()
		if elemType.Kind() != reflect.String {
			return fmt.Errorf("unsupported slice type: %s", field.Type())
		}
		cleaned := strings.Trim(value, "[]")
		if cleaned == "" {
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		} else {
			items := strings.Split(cleaned, ",")
			slice := reflect.MakeSlice(field.Type(), len(items), len(items))
			for i, item := range items {
				slice.Index(i).SetString(strings.TrimSpace(item))
			}
			field.Set(slice)
		}
	default:
		return fmt.Errorf("unsupported type: %s", field.Type())
	}
	return nil
}

// parseDuration parses a duration string (supports Go duration strings and simple numbers as seconds).
func parseDuration(s string) (int64, error) {
	// Try parsing as Go duration first
	if d, err := parseGoDuration(s); err == nil {
		return int64(d), nil
	}

	// Fallback to parsing as seconds
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	return n * 1e9, nil // Convert seconds to nanoseconds
}

// parseGoDuration parses Go-style duration strings like "15m", "1h30m", etc.
func parseGoDuration(s string) (int64, error) {
	// Simple duration parser for common formats
	var total int64
	var numStr string

	for _, ch := range s {
		switch ch {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			numStr += string(ch)
		case 's':
			if numStr == "" {
				return 0, fmt.Errorf("invalid duration")
			}
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", numStr)
			}
			total += n * 1e9 // seconds to nanoseconds
			numStr = ""
		case 'm':
			if numStr == "" {
				return 0, fmt.Errorf("invalid duration")
			}
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", numStr)
			}
			total += n * 60 * 1e9 // minutes to nanoseconds
			numStr = ""
		case 'h':
			if numStr == "" {
				return 0, fmt.Errorf("invalid duration")
			}
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", numStr)
			}
			total += n * 60 * 60 * 1e9 // hours to nanoseconds
			numStr = ""
		case 'd':
			if numStr == "" {
				return 0, fmt.Errorf("invalid duration")
			}
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", numStr)
			}
			total += n * 24 * 60 * 60 * 1e9 // days to nanoseconds
			numStr = ""
		default:
			return 0, fmt.Errorf("invalid duration unit: %c", ch)
		}
	}

	if numStr != "" {
		return 0, fmt.Errorf("incomplete duration")
	}

	return total, nil
}

// unmarshalYAML parses YAML data into the config struct.
// This is a minimal YAML parser sufficient for our config structure.
func unmarshalYAML(data []byte, cfg *Config) error {
	// For now, we'll use a simple line-by-line parser
	// In production, this would be replaced with a proper YAML parser
	// Since we want zero dependencies, we implement basic parsing

	lines := strings.Split(string(data), "\n")
	var currentPath []string
	var lastIndent int

	for _, line := range lines {
		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Calculate indentation
		indent := 0
		for i := 0; i < len(line) && (line[i] == ' ' || line[i] == '\t'); i++ {
			indent++
		}

		// Parse key-value pairs
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Adjust path based on indentation
			if indent <= lastIndent && len(currentPath) > 0 {
				levels := (lastIndent - indent) / 2
				if levels >= len(currentPath) {
					currentPath = nil
				} else {
					currentPath = currentPath[:len(currentPath)-levels-1]
				}
			}

			if value == "" {
				// This is a nested section
				currentPath = append(currentPath, key)
			} else {
				// This is a key-value pair
				fullPath := append(currentPath, key)
				if err := setField(cfg, fullPath, value); err != nil {
					// Silently skip unknown fields for forward compatibility
					continue
				}
			}

			lastIndent = indent
		}
	}

	return nil
}

// Save saves the configuration to a YAML file.
func Save(cfg *Config, path string) error {
	data, err := marshalYAML(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil { // #nosec G703 - path is validated before calling Save
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// marshalYAML converts the config struct to YAML format.
func marshalYAML(cfg *Config) ([]byte, error) {
	var lines []string

	lines = append(lines, "# VaultDrift Configuration File")
	lines = append(lines, "# Generated by interactive setup")
	lines = append(lines, "")

	// Server
	lines = append(lines, "server:")
	lines = append(lines, fmt.Sprintf("  host: %s", cfg.Server.Host))
	lines = append(lines, fmt.Sprintf("  port: %d", cfg.Server.Port))
	lines = append(lines, fmt.Sprintf("  base_url: %s", cfg.Server.BaseURL))
	lines = append(lines, "  tls:")
	lines = append(lines, fmt.Sprintf("    enabled: %t", cfg.Server.TLS.Enabled))
	lines = append(lines, fmt.Sprintf("    auto_cert: %t", cfg.Server.TLS.AutoCert))
	if cfg.Server.TLS.CertFile != "" {
		lines = append(lines, fmt.Sprintf("    cert_file: %s", cfg.Server.TLS.CertFile))
	}
	if cfg.Server.TLS.KeyFile != "" {
		lines = append(lines, fmt.Sprintf("    key_file: %s", cfg.Server.TLS.KeyFile))
	}

	// Storage
	lines = append(lines, "")
	lines = append(lines, "storage:")
	lines = append(lines, fmt.Sprintf("  backend: %s", cfg.Storage.Backend))
	lines = append(lines, "  local:")
	lines = append(lines, fmt.Sprintf("    data_dir: %s", cfg.Storage.Local.DataDir))
	if cfg.Storage.Backend == "s3" {
		lines = append(lines, "  s3:")
		lines = append(lines, fmt.Sprintf("    endpoint: %s", cfg.Storage.S3.Endpoint))
		lines = append(lines, fmt.Sprintf("    bucket: %s", cfg.Storage.S3.Bucket))
		lines = append(lines, fmt.Sprintf("    region: %s", cfg.Storage.S3.Region))
		lines = append(lines, fmt.Sprintf("    access_key: %s", cfg.Storage.S3.AccessKey))
		if cfg.Storage.S3.SecretKey != "" {
			lines = append(lines, "    secret_key: *****")
		}
	}

	// Database
	lines = append(lines, "")
	lines = append(lines, "database:")
	lines = append(lines, fmt.Sprintf("  path: %s", cfg.Database.Path))

	// Auth
	lines = append(lines, "")
	lines = append(lines, "auth:")
	lines = append(lines, "  jwt_secret: *****")
	lines = append(lines, fmt.Sprintf("  access_token_ttl: %s", formatDuration(cfg.Auth.AccessTokenTTL)))
	lines = append(lines, fmt.Sprintf("  refresh_token_ttl: %s", formatDuration(cfg.Auth.RefreshTokenTTL)))
	lines = append(lines, fmt.Sprintf("  totp_enabled: %t", cfg.Auth.TOTPEnabled))
	lines = append(lines, fmt.Sprintf("  max_login_attempts: %d", cfg.Auth.MaxLoginAttempts))
	lines = append(lines, fmt.Sprintf("  lockout_duration: %s", formatDuration(cfg.Auth.LockoutDuration)))

	// Sync
	lines = append(lines, "")
	lines = append(lines, "sync:")
	lines = append(lines, fmt.Sprintf("  chunk_size_min: %d", cfg.Sync.ChunkSizeMin))
	lines = append(lines, fmt.Sprintf("  chunk_size_avg: %d", cfg.Sync.ChunkSizeAvg))
	lines = append(lines, fmt.Sprintf("  chunk_size_max: %d", cfg.Sync.ChunkSizeMax))
	lines = append(lines, fmt.Sprintf("  max_concurrent_transfers: %d", cfg.Sync.MaxConcurrentTransfers))
	lines = append(lines, fmt.Sprintf("  websocket_enabled: %t", cfg.Sync.WebSocketEnabled))

	// Encryption
	lines = append(lines, "")
	lines = append(lines, "encryption:")
	lines = append(lines, fmt.Sprintf("  enabled: %t", cfg.Encryption.Enabled))
	lines = append(lines, fmt.Sprintf("  zero_knowledge: %t", cfg.Encryption.ZeroKnowledge))
	lines = append(lines, fmt.Sprintf("  argon2_time: %d", cfg.Encryption.Argon2Time))
	lines = append(lines, fmt.Sprintf("  argon2_memory: %d", cfg.Encryption.Argon2Memory))
	lines = append(lines, fmt.Sprintf("  argon2_threads: %d", cfg.Encryption.Argon2Threads))

	// Sharing
	lines = append(lines, "")
	lines = append(lines, "sharing:")
	lines = append(lines, fmt.Sprintf("  public_links_enabled: %t", cfg.Sharing.PublicLinksEnabled))
	lines = append(lines, fmt.Sprintf("  max_expiry_days: %d", cfg.Sharing.MaxExpiryDays))
	lines = append(lines, fmt.Sprintf("  default_expiry_days: %d", cfg.Sharing.DefaultExpiryDays))

	// Users
	lines = append(lines, "")
	lines = append(lines, "users:")
	lines = append(lines, fmt.Sprintf("  registration_enabled: %t", cfg.Users.RegistrationEnabled))
	lines = append(lines, fmt.Sprintf("  default_quota: %d", cfg.Users.DefaultQuota))

	// Logging
	lines = append(lines, "")
	lines = append(lines, "logging:")
	lines = append(lines, fmt.Sprintf("  level: %s", cfg.Logging.Level))
	lines = append(lines, fmt.Sprintf("  file: %s", cfg.Logging.File))
	lines = append(lines, fmt.Sprintf("  format: %s", cfg.Logging.Format))
	lines = append(lines, fmt.Sprintf("  audit: %t", cfg.Logging.Audit))

	// Federation
	lines = append(lines, "")
	lines = append(lines, "federation:")
	lines = append(lines, fmt.Sprintf("  enabled: %t", cfg.Federation.Enabled))

	return []byte(strings.Join(lines, "\n")), nil
}

// formatDuration converts a duration to a readable string
func formatDuration(d interface{}) string {
	switch v := d.(type) {
	case int64:
		secs := v / 1e9
		if secs >= 86400 {
			return fmt.Sprintf("%dd", secs/86400)
		}
		if secs >= 3600 {
			return fmt.Sprintf("%dh", secs/3600)
		}
		if secs >= 60 {
			return fmt.Sprintf("%dm", secs/60)
		}
		return fmt.Sprintf("%ds", secs)
	default:
		return "15m"
	}
}
