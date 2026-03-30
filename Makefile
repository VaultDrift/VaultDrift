VERSION    := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS    := -s -w -X main.version=$(VERSION)
GOOS       := $(shell go env GOOS)
GOARCH     := $(shell go env GOARCH)

.PHONY: build build-cli build-desktop build-all build-web test lint clean docker

# Default build target
build: build-server

# Server binary
build-server:
	@echo "Building VaultDrift server..."
	go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift ./cmd/vaultdrift

# CLI client
build-cli:
	@echo "Building VaultDrift CLI..."
	go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-cli ./cmd/vaultdrift-cli

# Desktop tray app
build-desktop:
	@echo "Building VaultDrift Desktop..."
	go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-desktop ./cmd/vaultdrift-desktop

# Web UI
build-web:
	@echo "Building Web UI..."
	cd web && npm run build

# Build all binaries for current platform
build-all: build-server build-cli build-desktop

# Cross-compile for all platforms
build-cross:
	@echo "Cross-compiling for all platforms..."
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-linux-amd64 ./cmd/vaultdrift
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-linux-arm64 ./cmd/vaultdrift
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-darwin-amd64 ./cmd/vaultdrift
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-darwin-arm64 ./cmd/vaultdrift
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-windows-amd64.exe ./cmd/vaultdrift

# Run tests
test:
	go test -race -coverprofile=coverage.out ./...

# Run integration tests
test-integration:
	go test -tags=integration -race ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./internal/chunk/ ./internal/crypto/ ./internal/sync/

# Lint code
lint:
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed, skipping..."; \
	fi

# Format code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out
	cd web && rm -rf dist/

# Build Docker image
docker:
	docker build -t vaultdrift/vaultdrift:$(VERSION) .

# Development server with hot reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi

# Install locally
install: build
	cp bin/vaultdrift $(GOPATH)/bin/
	cp bin/vaultdrift-cli $(GOPATH)/bin/

# Generate code (if needed)
generate:
	go generate ./...

# Check for vulnerabilities
vuln:
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

# Show help
help:
	@echo "VaultDrift Makefile targets:"
	@echo ""
	@echo "  build          - Build server binary (default)"
	@echo "  build-server   - Build server binary"
	@echo "  build-cli      - Build CLI client"
	@echo "  build-desktop  - Build desktop tray app"
	@echo "  build-web      - Build Web UI"
	@echo "  build-all      - Build all binaries for current platform"
	@echo "  build-cross    - Cross-compile for all platforms"
	@echo "  test           - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker         - Build Docker image"
	@echo "  dev            - Run development server with hot reload"
	@echo "  install        - Install binaries to GOPATH/bin"
	@echo "  generate       - Run code generation"
	@echo "  vuln           - Check for vulnerabilities"
	@echo "  help           - Show this help"
