# VaultDrift

> **Your Files. Your Vault. Your Drift.**

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-in%20progress-yellow.svg)]()

VaultDrift is a **zero-dependency, single-binary** self-hosted file sync and share solution. Built in pure Go with an embedded database, it replaces complex stacks like Nextcloud/ownCloud/Seafile with a single, fast, secure binary.

## Philosophy: #NOFORKANYMORE

- **Zero external dependencies** - Everything from S3 signing to TOTP is built from scratch
- **Single binary** - One file, complete functionality
- **Pure Go** - No CGo, cross-compile everywhere
- **End-to-end encryption** - Zero-knowledge architecture
- **Delta sync** - Only changed chunks transferred

## Features

| Feature | Status |
|---------|--------|
| Storage Abstraction (Local + S3) | ✅ |
| Content-Defined Chunking + Deduplication | ✅ |
| End-to-End Encryption (AES-256-GCM, X25519) | ✅ |
| Delta Sync Protocol | ✅ |
| Web UI (Vanilla JS) | ✅ |
| CLI Client with Sync | ✅ |
| Desktop Tray App | ✅ |
| Public Share Links | ✅ |
| TOTP 2FA | ✅ |
| RBAC Authorization | ✅ |
| Real-time Sync (WebSocket/SSE) | ✅ |
| File Versioning | ✅ |
| Trash/Recycle Bin | ✅ |

## Quick Start

### Installation

```bash
# Download binary (coming soon)
curl -fsSL https://vaultdrift.com/install.sh | sh

# Or build from source
git clone https://github.com/vaultdrift/vaultdrift
cd vaultdrift && make build
```

### First Run

```bash
# Initialize with admin user
vaultdrift init --admin-user admin --admin-email admin@example.com

# Start server
vaultdrift serve

# Or with custom config
vaultdrift serve --config /etc/vaultdrift/vaultdrift.yaml
```

### CLI Client

```bash
# Login
vaultdrift-cli login https://vault.example.com

# Sync folder
vaultdrift-cli init ~/VaultDrift
vaultdrift-cli daemon start

# One-time sync
vaultdrift-cli sync
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    VaultDrift Binary                     │
├─────────────────────────────────────────────────────────┤
│  HTTP/API │  WebDAV  │   Sync   │  Admin  │  Web UI    │
├─────────────────────────────────────────────────────────┤
│              Core Engine (VFS + Crypto)                  │
├─────────────────────────────────────────────────────────┤
│         Storage Abstraction (Local / S3)                 │
├─────────────────────────────────────────────────────────┤
│              CobaltDB (Embedded Database)                │
└─────────────────────────────────────────────────────────┘
```

## Development

### Prerequisites

- Go 1.23+
- Node.js 20+ (for Web UI)
- Make

### Building

```bash
# Build all binaries
make build-all

# Build with web UI
make build-web && make build

# Run tests
make test

# Run with hot reload
make dev
```

### Project Structure

```
vaultdrift/
├── cmd/
│   ├── vaultdrift/          # Server binary
│   ├── vaultdrift-cli/      # CLI client
│   └── vaultdrift-desktop/  # Desktop tray app
├── internal/
│   ├── server/              # HTTP server, middleware
│   ├── api/                 # REST API handlers
│   ├── webdav/              # WebDAV server
│   ├── vfs/                 # Virtual filesystem
│   ├── storage/             # Storage backends
│   ├── chunk/               # Content-defined chunking
│   ├── crypto/              # Encryption engine
│   ├── sync/                # Sync protocol
│   ├── auth/                # Authentication & RBAC
│   ├── share/               # Sharing engine
│   └── db/                  # CobaltDB integration
├── client/                  # Shared client library
├── desktop/                 # Desktop tray app
├── web/                     # React Web UI
└── cobaltdb/                # Embedded database
```

## Configuration

See `vaultdrift.yaml.example` for a complete configuration reference.

Environment variables override config file values:
```bash
VAULTDRIFT_SERVER_PORT=8443
VAULTDRIFT_STORAGE_BACKEND=s3
VAULTDRIFT_STORAGE_S3_BUCKET=mybucket
```

## Security

- **TLS 1.3** with automatic Let's Encrypt certificates
- **Argon2id** password hashing
- **AES-256-GCM** file encryption
- **X25519** key exchange for sharing
- **Vector clocks** for conflict detection
- Rate limiting, CSRF protection, audit logging

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

---

*VaultDrift — Zero dependencies. One binary. Complete control.*
