package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/federation"
	"github.com/vaultdrift/vaultdrift/internal/server"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/tracing"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
	"github.com/vaultdrift/vaultdrift/internal/worker"
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
	if err := os.MkdirAll(*dataDir, 0750); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Load or create config
	cfg, err := loadConfig(*configPath, *dataDir, *host, *port)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if errs := config.Validate(cfg); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("Config error: %v", e)
		}
		log.Fatalf("Fix configuration errors above and restart")
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

	// Initialize federation manager (disabled by default)
	fedCfg := federation.FederationConfig{
		Enabled:       cfg.Federation.Enabled,
		ServerID:      cfg.Federation.ServerID,
		PublicURL:     cfg.Federation.PublicURL,
		PrivateKey:    cfg.Federation.PrivateKey,
		PublicKey:     cfg.Federation.PublicKey,
		TrustedPeers:  cfg.Federation.TrustedPeers,
		AutoDiscovery: cfg.Federation.AutoDiscovery,
	}
	fedMgr, err := federation.NewManager(fedCfg, database)
	if err != nil {
		log.Printf("Failed to initialize federation: %v", err)
		fedMgr = nil
	}

	// Initialize tracing (if enabled)
	var tracingProvider *tracing.Provider
	if cfg.Tracing.Enabled {
		traceProvider, err := tracing.NewProvider(tracing.Config{
			Enabled:      true,
			Exporter:     cfg.Tracing.Exporter,
			OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
			ServiceName:  cfg.Tracing.ServiceName,
			SampleRate:   cfg.Tracing.SampleRate,
		})
		if err != nil {
			log.Printf("Failed to initialize tracing: %v", err)
		} else {
			tracingProvider = traceProvider
			log.Println("Tracing enabled:", cfg.Tracing.Exporter)
		}
	}

	// Initialize worker manager (background GC, trash cleanup)
	workerMgr := worker.NewManager(database, store)
	workerMgr.Start()
	defer workerMgr.Stop()

	// Create server
	srv := server.NewServer(cfg.Server, database, authSvc, vfsService, store, []byte(cfg.Auth.JWTSecret), fedMgr)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")

		// Shutdown tracing provider
		if tracingProvider != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := tracingProvider.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down tracing: %v", err)
			}
			cancel()
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Stop(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
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
		cfg, err := config.Load(configPath)
		if err == nil {
			return cfg, nil
		}
		// If file doesn't exist, we'll try to create one
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Check for config in data directory
	configFile := dataDir + "/config.yaml"
	if _, err := os.Stat(configFile); err == nil {
		return config.Load(configFile)
	}

	// No config found - run interactive setup
	fmt.Println("No configuration file found.")
	fmt.Println()

	cfg, err := config.InteractiveSetup()
	if err != nil {
		return nil, fmt.Errorf("interactive setup failed: %w", err)
	}

	return cfg, nil
}
