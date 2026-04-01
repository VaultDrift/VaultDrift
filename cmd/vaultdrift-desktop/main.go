package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/desktop"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to config file")
		dataDir    = flag.String("data", "", "Data directory (default: ~/.vaultdrift)")
		port       = flag.Int("port", 0, "Server port (default: 0 for random)")
	)
	flag.Parse()

	// Determine data directory
	if *dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}
		*dataDir = filepath.Join(homeDir, ".vaultdrift")
	}

	// Load or create config
	cfg, err := loadConfig(*configPath, *dataDir, *port)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create and run desktop app
	app, err := desktop.NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("App error: %v", err)
	}
}

// loadConfig loads or creates default configuration
func loadConfig(configPath, dataDir string, port int) (*config.Config, error) {
	// If config path specified, load from there
	if configPath != "" {
		return config.Load(configPath)
	}

	// Check for config in data directory
	configFile := filepath.Join(dataDir, "config.json")
	if _, err := os.Stat(configFile); err == nil {
		return config.Load(configFile)
	}

	// Create default config
	cfg := config.DefaultConfig()
	cfg.Database.Path = filepath.Join(dataDir, "vaultdrift.db")
	cfg.Storage.Local.DataDir = filepath.Join(dataDir, "storage")

	if port > 0 {
		cfg.Server.Port = port
	}

	// Ensure directories exist
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	if err := os.MkdirAll(cfg.Storage.Local.DataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return cfg, nil
}
