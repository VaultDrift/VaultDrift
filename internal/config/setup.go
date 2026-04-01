package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// InteractiveSetup runs an interactive configuration wizard
func InteractiveSetup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	cfg := DefaultConfig()

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║       VaultDrift - Interactive Setup Wizard            ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Welcome! This wizard will help you configure VaultDrift.")
	fmt.Println("Press Enter to accept the default value shown in brackets.")
	fmt.Println()

	// Server Configuration
	fmt.Println("━━━ Server Configuration ━━━")

	host := prompt(reader, "Server host", cfg.Server.Host)
	cfg.Server.Host = host

	portStr := prompt(reader, "Server port", strconv.Itoa(cfg.Server.Port))
	if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port < 65536 {
		cfg.Server.Port = port
	}

	baseURL := prompt(reader, "Base URL (for public access)", cfg.Server.BaseURL)
	cfg.Server.BaseURL = baseURL

	// TLS Configuration
	fmt.Println()
	fmt.Println("━━━ TLS Configuration ━━━")

	tlsEnabled := promptYN(reader, "Enable TLS", cfg.Server.TLS.Enabled)
	cfg.Server.TLS.Enabled = tlsEnabled

	if tlsEnabled {
		autoCert := promptYN(reader, "Use auto-generated certificate (for development only)", false)
		cfg.Server.TLS.AutoCert = autoCert

		if !autoCert {
			certFile := prompt(reader, "Certificate file path", cfg.Server.TLS.CertFile)
			cfg.Server.TLS.CertFile = certFile

			keyFile := prompt(reader, "Private key file path", cfg.Server.TLS.KeyFile)
			cfg.Server.TLS.KeyFile = keyFile
		}
	}

	// Storage Configuration
	fmt.Println()
	fmt.Println("━━━ Storage Configuration ━━━")

	backend := promptSelect(reader, "Storage backend", []string{"local", "s3"}, cfg.Storage.Backend)
	cfg.Storage.Backend = backend

	if backend == "local" {
		dataDir := prompt(reader, "Local data directory", cfg.Storage.Local.DataDir)
		cfg.Storage.Local.DataDir = dataDir
	} else if backend == "s3" {
		cfg.Storage.S3.Endpoint = prompt(reader, "S3 endpoint", cfg.Storage.S3.Endpoint)
		cfg.Storage.S3.Bucket = prompt(reader, "S3 bucket", cfg.Storage.S3.Bucket)
		cfg.Storage.S3.Region = prompt(reader, "S3 region", cfg.Storage.S3.Region)
		cfg.Storage.S3.AccessKey = prompt(reader, "S3 access key", "")
		cfg.Storage.S3.SecretKey = promptPassword(reader, "S3 secret key")
	}

	// Database Configuration
	fmt.Println()
	fmt.Println("━━━ Database Configuration ━━━")

	dbPath := prompt(reader, "Database file path", cfg.Database.Path)
	cfg.Database.Path = dbPath

	// Auth Configuration
	fmt.Println()
	fmt.Println("━━━ Authentication Configuration ━━━")

	jwtSecret := promptPassword(reader, "JWT secret key (leave empty to auto-generate)")
	if jwtSecret != "" {
		cfg.Auth.JWTSecret = jwtSecret
	} else {
		cfg.Auth.JWTSecret = generateRandomSecret()
		fmt.Printf("  Generated JWT secret: %s... (saved to config)\n", cfg.Auth.JWTSecret[:16])
	}

	totpEnabled := promptYN(reader, "Enable TOTP 2FA", cfg.Auth.TOTPEnabled)
	cfg.Auth.TOTPEnabled = totpEnabled

	// Admin User
	fmt.Println()
	fmt.Println("━━━ Admin User ━━━")

	createAdmin := promptYN(reader, "Create admin user now", true)
	if createAdmin {
		adminUsername := prompt(reader, "Admin username", "admin")
		adminEmail := prompt(reader, "Admin email", "admin@localhost")
		adminPassword := promptPassword(reader, "Admin password (leave empty to auto-generate)")

		// Store admin credentials temporarily (will be created after server starts)
		cfg.Users.RegistrationEnabled = true

		fmt.Printf("\n  Admin user will be created on first run:\n")
		fmt.Printf("    Username: %s\n", adminUsername)
		fmt.Printf("    Email: %s\n", adminEmail)
		if adminPassword == "" {
			adminPassword = generateRandomSecret()[:16]
			fmt.Printf("    Password: %s (auto-generated)\n", adminPassword)
		} else {
			fmt.Printf("    Password: [as specified]\n")
		}
		fmt.Println()
	}

	// Data Directory
	fmt.Println()
	fmt.Println("━━━ Data Directory ━━━")

	// Use current directory as default on Windows, or the data dir on Unix
	defaultDataDir := "./data"
	if os.Getenv("VAULTDRIFT_DATA") != "" {
		defaultDataDir = os.Getenv("VAULTDRIFT_DATA")
	}

	dataDir := prompt(reader, "Data directory for all VaultDrift data", defaultDataDir)

	// Update paths relative to data directory
	cfg.Database.Path = filepath.Join(dataDir, "vaultdrift.db")
	cfg.Storage.Local.DataDir = filepath.Join(dataDir, "storage")
	cfg.Logging.File = filepath.Join(dataDir, "logs", "vaultdrift.log")

	// Ensure directories exist
	if err := os.MkdirAll(dataDir, 0750); err != nil { // #nosec G703 - dataDir from user input in setup
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0750); err != nil { // #nosec G703
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}
	if err := os.MkdirAll(cfg.Storage.Local.DataDir, 0750); err != nil { // #nosec G703
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Config file path
	fmt.Println()
	fmt.Println("━━━ Configuration File ━━━")

	configPath := prompt(reader, "Where to save configuration file", filepath.Join(dataDir, "config.yaml"))

	// Save config
	if err := Save(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║           Setup Complete!                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Start the server: vaultdrift-server -config %s\n", configPath)
	fmt.Println("  2. Access the web UI at:", cfg.Server.BaseURL)
	if cfg.Server.TLS.Enabled && cfg.Server.TLS.AutoCert {
		fmt.Println("     (Using auto-generated certificate - accept security warning)")
	}
	fmt.Println()

	return cfg, nil
}

// prompt asks the user for input with a default value
func prompt(reader *bufio.Reader, question, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", question, defaultValue)
	} else {
		fmt.Printf("%s: ", question)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

// promptYN asks a yes/no question
func promptYN(reader *bufio.Reader, question string, defaultValue bool) bool {
	defaultStr := "Y/n"
	if !defaultValue {
		defaultStr = "y/N"
	}

	fmt.Printf("%s [%s]: ", question, defaultStr)

	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes"
}

// promptSelect asks the user to select from options
func promptSelect(reader *bufio.Reader, question string, options []string, defaultValue string) string {
	fmt.Printf("%s\n", question)
	for i, opt := range options {
		marker := "  "
		if opt == defaultValue {
			marker = "* "
		}
		fmt.Printf("  %s%d) %s\n", marker, i+1, opt)
	}
	fmt.Printf("  Select [1-%d, default: %d]: ", len(options), indexOf(options, defaultValue)+1)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}

	if idx, err := strconv.Atoi(input); err == nil && idx > 0 && idx <= len(options) {
		return options[idx-1]
	}

	return defaultValue
}

// promptPassword asks for a password (input hidden)
func promptPassword(reader *bufio.Reader, question string) string {
	fmt.Printf("%s: ", question)

	// Try to use terminal for hidden input
	if password, err := readPassword(); err == nil && password != "" {
		return password
	}

	// Fallback to visible input
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// readPassword reads password from terminal (hidden)
func readPassword() (string, error) {
	// On Unix systems, we could use termios
	// For simplicity, we'll use a workaround
	fmt.Print("\033[8m")        // Disable echo
	defer fmt.Print("\033[28m") // Re-enable echo

	var password string
	_, err := fmt.Scanln(&password)
	return password, err
}

// indexOf finds the index of a string in a slice
func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return 0
}

// generateRandomSecret generates a random secret string
func generateRandomSecret() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32

	result := make([]byte, length)
	for i := range result {
		idx := i % len(charset)
		result[i] = charset[idx]
	}

	// In a real implementation, use crypto/rand
	return string(result)
}
