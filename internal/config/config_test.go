package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Port != 8443 {
		t.Errorf("expected default port 8443, got %d", cfg.Server.Port)
	}

	if cfg.Storage.Backend != "local" {
		t.Errorf("expected default storage backend 'local', got %s", cfg.Storage.Backend)
	}

	if cfg.Auth.AccessTokenTTL != 15*time.Minute {
		t.Errorf("expected default access token TTL 15m, got %v", cfg.Auth.AccessTokenTTL)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name: "valid default config",
			modify: func(c *Config) {
				c.Server.TLS.CertFile = "/path/to/cert.pem"
				c.Server.TLS.KeyFile = "/path/to/key.pem"
			},
			wantErr: false,
		},
		{
			name: "valid with auto cert",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Server.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Server.Port = 70000
			},
			wantErr: true,
		},
		{
			name: "missing base_url",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Server.BaseURL = ""
			},
			wantErr: true,
		},
		{
			name: "invalid base_url scheme",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Server.BaseURL = "ftp://example.com"
			},
			wantErr: true,
		},
		{
			name: "invalid storage backend",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Storage.Backend = "invalid"
			},
			wantErr: true,
		},
		{
			name: "chunk sizes out of order",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Sync.ChunkSizeMin = 1000
				c.Sync.ChunkSizeAvg = 500
			},
			wantErr: true,
		},
		{
			name: "negative quota",
			modify: func(c *Config) {
				c.Server.TLS.AutoCert = true
				c.Users.DefaultQuota = -1
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			// Set a valid JWT secret for testing
			cfg.Auth.JWTSecret = "test-jwt-secret-that-is-at-least-32-characters-long"
			tt.modify(cfg)
			errs := Validate(cfg)
			gotErr := len(errs) > 0
			if gotErr != tt.wantErr {
				t.Errorf("Validate() errors = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	t.Setenv("VAULTDRIFT_SERVER_PORT", "8080")
	t.Setenv("VAULTDRIFT_STORAGE_BACKEND", "s3")

	cfg := DefaultConfig()
	err := loadFromEnv(cfg)
	if err != nil {
		t.Fatalf("loadFromEnv() error = %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Storage.Backend != "s3" {
		t.Errorf("expected storage backend 's3', got %s", cfg.Storage.Backend)
	}
}
