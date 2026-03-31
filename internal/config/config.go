// Package config provides configuration management for VaultDrift.
// Configuration is loaded from YAML files and can be overridden via environment variables.
package config

import (
	"time"
)

// Config holds all configuration for the VaultDrift server.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Storage    StorageConfig    `yaml:"storage"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Sync       SyncConfig       `yaml:"sync"`
	Encryption EncryptionConfig `yaml:"encryption"`
	Sharing    SharingConfig    `yaml:"sharing"`
	Users      UsersConfig      `yaml:"users"`
	SMTP       SMTPConfig       `yaml:"smtp"`
	Logging    LoggingConfig    `yaml:"logging"`
	Federation FederationConfig `yaml:"federation"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host    string    `yaml:"host"`
	Port    int       `yaml:"port"`
	TLS     TLSConfig `yaml:"tls"`
	BaseURL string    `yaml:"base_url"`
}

// TLSConfig holds TLS certificate configuration.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	AutoCert bool   `yaml:"auto_cert"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// StorageConfig holds storage backend configuration.
type StorageConfig struct {
	Backend string      `yaml:"backend"` // "local", "s3", or "ipfs"
	Local   LocalConfig `yaml:"local"`
	S3      S3Config    `yaml:"s3"`
	IPFS    IPFSConfig  `yaml:"ipfs"`
}

// LocalConfig holds local filesystem storage configuration.
type LocalConfig struct {
	DataDir string `yaml:"data_dir"`
}

// S3Config holds S3-compatible storage configuration.
type S3Config struct {
	Endpoint     string `yaml:"endpoint"`
	Bucket       string `yaml:"bucket"`
	Region       string `yaml:"region"`
	AccessKey    string `yaml:"access_key"`
	SecretKey    string `yaml:"secret_key"`
	UsePathStyle bool   `yaml:"use_path_style"`
}

// IPFSConfig holds IPFS storage configuration.
type IPFSConfig struct {
	APIAddr  string `yaml:"api_addr"`  // Multiaddr of IPFS API
	Gateway  string `yaml:"gateway"`   // IPFS Gateway URL
	PinFiles bool   `yaml:"pin_files"` // Whether to pin stored files
}

// DatabaseConfig holds CobaltDB configuration.
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret        string        `yaml:"jwt_secret"`
	AccessTokenTTL   time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `yaml:"refresh_token_ttl"`
	TOTPEnabled      bool          `yaml:"totp_enabled"`
	MaxLoginAttempts int           `yaml:"max_login_attempts"`
	LockoutDuration  time.Duration `yaml:"lockout_duration"`
}

// SyncConfig holds sync protocol configuration.
type SyncConfig struct {
	ChunkSizeMin           int  `yaml:"chunk_size_min"`
	ChunkSizeAvg           int  `yaml:"chunk_size_avg"`
	ChunkSizeMax           int  `yaml:"chunk_size_max"`
	MaxConcurrentTransfers int  `yaml:"max_concurrent_transfers"`
	WebSocketEnabled       bool `yaml:"websocket_enabled"`
}

// EncryptionConfig holds encryption configuration.
type EncryptionConfig struct {
	Enabled       bool   `yaml:"enabled"`
	ZeroKnowledge bool   `yaml:"zero_knowledge"`
	Argon2Time    uint32 `yaml:"argon2_time"`
	Argon2Memory  uint32 `yaml:"argon2_memory"`
	Argon2Threads uint8  `yaml:"argon2_threads"`
}

// SharingConfig holds sharing configuration.
type SharingConfig struct {
	PublicLinksEnabled bool `yaml:"public_links_enabled"`
	MaxExpiryDays      int  `yaml:"max_expiry_days"`
	DefaultExpiryDays  int  `yaml:"default_expiry_days"`
	PasswordRequired   bool `yaml:"password_required"`
	MaxDownloadLimit   int  `yaml:"max_download_limit"`
}

// UsersConfig holds user management configuration.
type UsersConfig struct {
	RegistrationEnabled bool  `yaml:"registration_enabled"`
	DefaultQuota        int64 `yaml:"default_quota"`
	MaxQuota            int64 `yaml:"max_quota"`
}

// SMTPConfig holds SMTP configuration for notifications.
type SMTPConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	TLS      bool   `yaml:"tls"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `yaml:"level"` // debug, info, warn, error
	File   string `yaml:"file"`
	Audit  bool   `yaml:"audit"`
	Format string `yaml:"format"` // json, text
}

// FederationConfig holds federation configuration.
type FederationConfig struct {
	Enabled       bool     `yaml:"enabled"`
	ServerID      string   `yaml:"server_id"`
	PublicURL     string   `yaml:"public_url"`
	PrivateKey    string   `yaml:"private_key"`
	PublicKey     string   `yaml:"public_key"`
	TrustedPeers  []string `yaml:"trusted_peers"`
	AutoDiscovery bool     `yaml:"auto_discovery"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    "0.0.0.0",
			Port:    8443,
			BaseURL: "https://localhost:8443",
			TLS: TLSConfig{
				Enabled:  true,
				AutoCert: false,
			},
		},
		Storage: StorageConfig{
			Backend: "local",
			Local: LocalConfig{
				DataDir: "/var/lib/vaultdrift/data",
			},
			S3: S3Config{
				Region:       "us-east-1",
				UsePathStyle: false,
			},
		},
		Database: DatabaseConfig{
			Path: "/var/lib/vaultdrift/db/vaultdrift.cdb",
		},
		Auth: AuthConfig{
			AccessTokenTTL:   15 * time.Minute,
			RefreshTokenTTL:  7 * 24 * time.Hour,
			TOTPEnabled:      true,
			MaxLoginAttempts: 5,
			LockoutDuration:  15 * time.Minute,
		},
		Sync: SyncConfig{
			ChunkSizeMin:           256 * 1024,      // 256KB
			ChunkSizeAvg:           1024 * 1024,     // 1MB
			ChunkSizeMax:           4 * 1024 * 1024, // 4MB
			MaxConcurrentTransfers: 4,
			WebSocketEnabled:       true,
		},
		Encryption: EncryptionConfig{
			Enabled:       true,
			ZeroKnowledge: true,
			Argon2Time:    3,
			Argon2Memory:  64 * 1024, // 64MB
			Argon2Threads: 4,
		},
		Sharing: SharingConfig{
			PublicLinksEnabled: true,
			MaxExpiryDays:      90,
			DefaultExpiryDays:  7,
			PasswordRequired:   false,
			MaxDownloadLimit:   1000,
		},
		Users: UsersConfig{
			RegistrationEnabled: false,
			DefaultQuota:        10 * 1024 * 1024 * 1024, // 10GB
			MaxQuota:            0,                       // 0 = unlimited
		},
		SMTP: SMTPConfig{
			Enabled: false,
			Port:    587,
			TLS:     true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			File:   "/var/lib/vaultdrift/logs/vaultdrift.log",
			Audit:  true,
			Format: "json",
		},
		Federation: FederationConfig{
			Enabled:       false, // Disabled by default
			AutoDiscovery: false,
			TrustedPeers:  []string{},
		},
	}
}
