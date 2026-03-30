package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate checks if the configuration is valid.
// Returns a slice of validation errors (empty if valid).
func Validate(cfg *Config) []error {
	var errs []error

	// Server validation
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535"))
	}

	if cfg.Server.BaseURL == "" {
		errs = append(errs, fmt.Errorf("server.base_url is required"))
	}

	if !strings.HasPrefix(cfg.Server.BaseURL, "http://") && !strings.HasPrefix(cfg.Server.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("server.base_url must start with http:// or https://"))
	}

	if cfg.Server.TLS.Enabled && !cfg.Server.TLS.AutoCert {
		if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
			errs = append(errs, fmt.Errorf("server.tls.cert_file and server.tls.key_file are required when auto_cert is disabled"))
		}
	}

	// Storage validation
	if cfg.Storage.Backend != "local" && cfg.Storage.Backend != "s3" {
		errs = append(errs, fmt.Errorf("storage.backend must be 'local' or 's3'"))
	}

	if cfg.Storage.Backend == "local" {
		if cfg.Storage.Local.DataDir == "" {
			errs = append(errs, fmt.Errorf("storage.local.data_dir is required"))
		}
	}

	if cfg.Storage.Backend == "s3" {
		if cfg.Storage.S3.Bucket == "" {
			errs = append(errs, fmt.Errorf("storage.s3.bucket is required"))
		}
		if cfg.Storage.S3.Region == "" {
			errs = append(errs, fmt.Errorf("storage.s3.region is required"))
		}
	}

	// Database validation
	if cfg.Database.Path == "" {
		errs = append(errs, fmt.Errorf("database.path is required"))
	}

	// Auth validation
	if cfg.Auth.AccessTokenTTL <= 0 {
		errs = append(errs, fmt.Errorf("auth.access_token_ttl must be positive"))
	}

	if cfg.Auth.RefreshTokenTTL <= 0 {
		errs = append(errs, fmt.Errorf("auth.refresh_token_ttl must be positive"))
	}

	if cfg.Auth.MaxLoginAttempts < 1 {
		errs = append(errs, fmt.Errorf("auth.max_login_attempts must be at least 1"))
	}

	// Sync validation
	if cfg.Sync.ChunkSizeMin < 1024 {
		errs = append(errs, fmt.Errorf("sync.chunk_size_min must be at least 1KB"))
	}

	if cfg.Sync.ChunkSizeAvg < cfg.Sync.ChunkSizeMin {
		errs = append(errs, fmt.Errorf("sync.chunk_size_avg must be >= chunk_size_min"))
	}

	if cfg.Sync.ChunkSizeMax < cfg.Sync.ChunkSizeAvg {
		errs = append(errs, fmt.Errorf("sync.chunk_size_max must be >= chunk_size_avg"))
	}

	if cfg.Sync.MaxConcurrentTransfers < 1 {
		errs = append(errs, fmt.Errorf("sync.max_concurrent_transfers must be at least 1"))
	}

	// Encryption validation
	if cfg.Encryption.Argon2Time < 1 {
		errs = append(errs, fmt.Errorf("encryption.argon2_time must be at least 1"))
	}

	if cfg.Encryption.Argon2Memory < 8*1024 {
		errs = append(errs, fmt.Errorf("encryption.argon2_memory must be at least 8MB"))
	}

	if cfg.Encryption.Argon2Threads < 1 {
		errs = append(errs, fmt.Errorf("encryption.argon2_threads must be at least 1"))
	}

	// Sharing validation
	if cfg.Sharing.MaxExpiryDays < 1 {
		errs = append(errs, fmt.Errorf("sharing.max_expiry_days must be at least 1"))
	}

	if cfg.Sharing.DefaultExpiryDays < 1 {
		errs = append(errs, fmt.Errorf("sharing.default_expiry_days must be at least 1"))
	}

	if cfg.Sharing.DefaultExpiryDays > cfg.Sharing.MaxExpiryDays {
		errs = append(errs, fmt.Errorf("sharing.default_expiry_days must be <= max_expiry_days"))
	}

	// Users validation
	if cfg.Users.DefaultQuota < 0 {
		errs = append(errs, fmt.Errorf("users.default_quota must be non-negative"))
	}

	if cfg.Users.MaxQuota < 0 {
		errs = append(errs, fmt.Errorf("users.max_quota must be non-negative (0 = unlimited)"))
	}

	if cfg.Users.MaxQuota > 0 && cfg.Users.DefaultQuota > cfg.Users.MaxQuota {
		errs = append(errs, fmt.Errorf("users.default_quota must be <= max_quota"))
	}

	// SMTP validation (only if enabled)
	if cfg.SMTP.Enabled {
		if cfg.SMTP.Host == "" {
			errs = append(errs, fmt.Errorf("smtp.host is required when smtp is enabled"))
		}
		if cfg.SMTP.Port < 1 || cfg.SMTP.Port > 65535 {
			errs = append(errs, fmt.Errorf("smtp.port must be between 1 and 65535"))
		}
		if cfg.SMTP.From == "" {
			errs = append(errs, fmt.Errorf("smtp.from is required when smtp is enabled"))
		}
	}

	// Logging validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.Logging.Level] {
		errs = append(errs, fmt.Errorf("logging.level must be one of: debug, info, warn, error"))
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[cfg.Logging.Format] {
		errs = append(errs, fmt.Errorf("logging.format must be one of: json, text"))
	}

	return errs
}

// ValidateAndExit validates the config and exits with an error message if invalid.
func ValidateAndExit(cfg *Config) {
	if errs := Validate(cfg); len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "Configuration errors:")
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", err)
		}
		os.Exit(1)
	}
}

// EnsureDirs creates necessary directories if they don't exist.
func EnsureDirs(cfg *Config) error {
	dirs := []string{}

	// Data directory
	if cfg.Storage.Backend == "local" {
		dirs = append(dirs, cfg.Storage.Local.DataDir)
		dirs = append(dirs, filepath.Join(cfg.Storage.Local.DataDir, "chunks"))
	}

	// Database directory
	dbDir := filepath.Dir(cfg.Database.Path)
	if dbDir != "" && dbDir != "." {
		dirs = append(dirs, dbDir)
	}

	// Log directory
	logDir := filepath.Dir(cfg.Logging.File)
	if logDir != "" && logDir != "." {
		dirs = append(dirs, logDir)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// IsDevelopment returns true if running in development mode.
func IsDevelopment(cfg *Config) bool {
	return cfg.Server.BaseURL == "" ||
		strings.Contains(cfg.Server.BaseURL, "localhost") ||
		strings.Contains(cfg.Server.BaseURL, "127.0.0.1")
}
