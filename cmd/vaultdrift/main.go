package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/server"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to config file")
		dataDir    = flag.String("data", "./data", "Data directory")
		port       = flag.Int("port", 8080, "Server port")
		host       = flag.String("host", "0.0.0.0", "Server host")
	)
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Load or create config
	cfg, err := loadConfig(*configPath, *dataDir, *host, *port)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	database, err := db.Open(db.Config{Path: cfg.Database.Path})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Initialize storage backend
	store, err := storage.NewBackend(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize VFS
	vfsService := vfs.NewVFS(database)

	// Initialize auth service
	authSvc := auth.NewService(database, []byte(cfg.Auth.JWTSecret))

	// Create server
	srv := server.NewServer(cfg.Server, database, authSvc, vfsService, store, []byte(cfg.Auth.JWTSecret))

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		srv.Stop(nil)
	}()

	// Start server
	log.Printf("VaultDrift server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadConfig(configPath, dataDir, host string, port int) (*config.Config, error) {
	// If config path specified, load from there
	if configPath != "" {
		return config.Load(configPath)
	}

	// Check for config in data directory
	configFile := dataDir + "/config.yaml"
	if _, err := os.Stat(configFile); err == nil {
		return config.Load(configFile)
	}

	// Create default config
	cfg := config.DefaultConfig()
	cfg.Server.Host = host
	cfg.Server.Port = port
	cfg.Database.Path = dataDir + "/vaultdrift.db"
	cfg.Storage.Backend = "local"
	cfg.Storage.Local.DataDir = dataDir + "/storage"

	// Ensure storage directory exists
	os.MkdirAll(cfg.Storage.Local.DataDir, 0755)

	return cfg, nil
}
