package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds CLI configuration
type Config struct {
	ServerURL  string `json:"server_url"`
	Token      string `json:"token,omitempty"`
	Username   string `json:"username,omitempty"`
	DefaultDir string `json:"default_dir,omitempty"`
}

// ConfigManager handles configuration storage and retrieval
type ConfigManager struct {
	configDir  string
	configFile string
}

// NewConfigManager creates a new config manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".vaultdrift")
	return &ConfigManager{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.json"),
	}, nil
}

// Load loads the configuration from disk
func (cm *ConfigManager) Load() (*Config, error) {
	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config
			return &Config{
				ServerURL: "http://localhost:8080",
			}, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return &config, nil
}

// Save saves the configuration to disk
func (cm *ConfigManager) Save(config *Config) error {
	// Ensure config directory exists
	if err := os.MkdirAll(cm.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configFile, data, 0600)
}

// GetConfigDir returns the configuration directory path
func (cm *ConfigManager) GetConfigDir() string {
	return cm.configDir
}

// PromptInput prompts the user for input
func PromptInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// PromptPassword prompts the user for a password (hidden input)
func PromptPassword(prompt string) string {
	// On Windows, we can't easily hide input without external deps
	// So we just read normally
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
