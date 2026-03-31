VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS    := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)
GOOS       := $(shell go env GOOS)
GOARCH     := $(shell go env GOARCH)

.PHONY: build build-cli build-desktop build-all build-web test lint clean docker help

# Default build target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "VaultDrift Build System"
	@echo "======================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: build-web ## Build server binary for current platform (default)
	@echo "Building VaultDrift server $(VERSION)..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-server ./cmd/server

build-cli: ## Build CLI client binary
	@echo "Building VaultDrift CLI $(VERSION)..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-cli ./cmd/vaultdrift-cli

build-desktop: ## Build desktop tray app binary
	@echo "Building VaultDrift Desktop $(VERSION)..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-desktop ./cmd/vaultdrift-desktop

build-web: ## Build web UI
	@echo "Building Web UI..."
	cd web && npm install && npm run build

build-all: build-web ## Build all binaries for current platform
	@echo "Building all binaries..."
	$(MAKE) build build-cli build-desktop

build-cross: build-web ## Cross-compile for all platforms
	@echo "Cross-compiling for all platforms..."
	mkdir -p dist
	# Server binaries
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-server-linux-amd64 ./cmd/server
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-server-linux-arm64 ./cmd/server
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-server-darwin-amd64 ./cmd/server
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-server-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-server-windows-amd64.exe ./cmd/server
	# CLI binaries
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-cli-linux-amd64 ./cmd/vaultdrift-cli
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-cli-darwin-amd64 ./cmd/vaultdrift-cli
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/vaultdrift-cli-windows-amd64.exe ./cmd/vaultdrift-cli
	@echo "Cross-compile complete! Binaries in ./dist/"

test: ## Run all tests
	@echo "Running tests..."
	go test -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage report
	@echo "Coverage report:"
	go tool cover -func=coverage.out

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -tags=integration -race ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./internal/chunk/... ./internal/crypto/... ./internal/sync/...

lint: ## Run linters
	@echo "Running linters..."
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed, skipping..."; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/ dist/ coverage.out
	rm -rf web/dist/

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t vaultdrift/vaultdrift:$(VERSION) -t vaultdrift/vaultdrift:latest .

docker-up: ## Start with Docker Compose
	@echo "Starting with Docker Compose..."
	docker-compose up -d

docker-down: ## Stop Docker Compose
	@echo "Stopping Docker Compose..."
	docker-compose down

run: build ## Build and run server
	./bin/vaultdrift-server serve

dev: ## Run development server
	@echo "Starting development server..."
	go run ./cmd/server serve --dev

dev-web: ## Run web UI development server
	@echo "Starting web UI dev server..."
	cd web && npm run dev

install: build build-cli ## Install binaries to GOPATH/bin
	@echo "Installing binaries..."
	cp bin/vaultdrift-server $(GOPATH)/bin/ 2>/dev/null || cp bin/vaultdrift-server ~/go/bin/
	cp bin/vaultdrift-cli $(GOPATH)/bin/ 2>/dev/null || cp bin/vaultdrift-cli ~/go/bin/

generate: ## Generate code (mocks, etc.)
	@echo "Generating code..."
	go generate ./...

vuln: ## Check for vulnerabilities
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi
