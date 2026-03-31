#!/bin/bash
set -e

# Build script for VaultDrift

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS="-w -s \
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.Commit=${COMMIT}"

echo "Building VaultDrift ${VERSION}..."

# Build web UI
echo "Building web UI..."
cd web
npm install
npm run build
cd ..

# Build server binaries
echo "Building server binaries..."

# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-server-linux-amd64 ./cmd/server

# Linux ARM64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-server-linux-arm64 ./cmd/server

# macOS AMD64
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-server-darwin-amd64 ./cmd/server

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-server-darwin-arm64 ./cmd/server

# Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-server-windows-amd64.exe ./cmd/server

# Build CLI binaries
echo "Building CLI binaries..."

# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-cli-linux-amd64 ./cmd/vaultdrift-cli

# macOS AMD64
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-cli-darwin-amd64 ./cmd/vaultdrift-cli

# Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o dist/vaultdrift-cli-windows-amd64.exe ./cmd/vaultdrift-cli

echo "Build complete! Binaries in ./dist/"
