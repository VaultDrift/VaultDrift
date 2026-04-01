# VaultDrift

> **Your Files. Your Vault. Your Drift.**

[![CI](https://github.com/VaultDrift/VaultDrift/actions/workflows/ci.yml/badge.svg)](https://github.com/VaultDrift/VaultDrift/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.1.0--beta.1-orange.svg)]()
[![Website](https://img.shields.io/badge/website-vaultdrift.com-blue.svg)](https://vaultdrift.com)

⚠️ **Beta Status**: VaultDrift is currently in beta. APIs and features may change.

VaultDrift is a **secure, distributed file storage system** with end-to-end encryption, content-defined chunking, and real-time synchronization. Built in pure Go with an embedded SQLite database, it provides a complete self-hosted cloud storage solution.

🌐 **Website**: [vaultdrift.com](https://vaultdrift.com)

## Philosophy: #NOFORKANYMORE

- **Zero external dependencies** - Everything from S3 signing to TOTP is built from scratch
- **Single binary** - One file, complete functionality
- **End-to-end encryption** - Zero-knowledge architecture with AES-256-GCM
- **Delta sync** - Only changed chunks transferred using Rabin CDC
- **Real-time collaboration** - WebSocket-based live sync

## Features

| Feature | Status | Description |
|---------|--------|-------------|
| Storage Abstraction (Local + S3) | ✅ | Pluggable storage backends |
| Content-Defined Chunking | ✅ | Rabin fingerprinting, 256KB-4MB chunks |
| Deduplication | ✅ | Global block-level deduplication |
| End-to-End Encryption | ✅ | AES-256-GCM, X25519 key exchange |
| Delta Sync Protocol | ✅ | Transfer only changed chunks |
| Vector Clocks | ✅ | Distributed conflict resolution |
| Merkle Trees | ✅ | Efficient sync state comparison |
| Web UI (React 19) | ✅ | Modern SPA with Tailwind CSS |
| CLI Client | ✅ | Full-featured command-line client |
| Desktop Tray App | ✅ | System tray with auto-sync |
| WebDAV Server | ✅ | Class 2 compliant WebDAV |
| Public Share Links | ✅ | Expiring links with passwords |
| TOTP 2FA | ✅ | Time-based one-time passwords |
| RBAC Authorization | ✅ | Role-based access control |
| Real-time Sync | ✅ | WebSocket event broadcasting |
| File Versioning | ✅ | Keep multiple versions |
| Trash/Recycle Bin | ✅ | 30-day retention |
| Thumbnail Generation | ✅ | Async image thumbnails |
| Background Workers | ✅ | GC, cleanup, maintenance |

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+ and npm (for web UI)
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/vaultdrift/vaultdrift.git
cd vaultdrift

# Install dependencies
go mod download
cd web && npm install && cd ..

# Build the web UI
cd web && npm run build && cd ..

# Build binaries (requires CGO for SQLite)
go build -o vaultdrift-server ./cmd/vaultdrift
go build -o vaultdrift-cli ./cmd/vaultdrift-cli
```

### Running the Server

```bash
# Initialize with admin user
./vaultdrift-server init --admin-user admin --admin-email admin@example.com

# Start server
./vaultdrift-server serve

# Or with custom config
./vaultdrift-server serve --config /etc/vaultdrift/config.yaml
```

Default server URL: `https://localhost:8443`

### Using the CLI Client

```bash
# Configure server
./vaultdrift-cli config server https://vault.example.com

# Login
./vaultdrift-cli login

# List files
./vaultdrift-cli ls

# Upload a file
./vaultdrift-cli upload ./document.pdf

# Download a file
./vaultdrift-cli download document.pdf

# Create a share link
./vaultdrift-cli share document.pdf --expires 7

# Sync a folder
./vaultdrift-cli sync ./my-folder

# Run sync daemon (auto-upload on changes)
./vaultdrift-cli daemon ./my-folder
```

### Using the Desktop App

> ⚠️ Desktop app is currently Linux-only due to systray/CGO requirements.

```bash
# Run the desktop tray application (Linux only)
go build -o vaultdrift-desktop ./cmd/vaultdrift-desktop
./vaultdrift-desktop

# The app will appear in your system tray
# Click to open the web interface in your browser
```

### Using WebDAV

Mount VaultDrift as a network drive:

```bash
# Linux (davfs2)
mount -t davfs https://vault.example.com/webdav /mnt/vaultdrift

# macOS
mount_webdav https://vault.example.com/webdav /Volumes/VaultDrift

# Windows
net use Z: https://vault.example.com/webdav
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Clients                                 │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │
│  │   Web UI   │ │ CLI Client │ │   Desktop  │ │   WebDAV   │   │
│  │  (React)   │ │   (Go)     │ │  (Tray)    │ │   Client   │   │
│  └──────┬─────┘ └──────┬─────┘ └──────┬─────┘ └──────┬─────┘   │
│         │              │              │              │          │
│         └──────────────┴──────┬───────┴──────────────┘          │
│                               │                                  │
│                    HTTP / WebSocket / WebDAV                     │
└───────────────────────────────┬──────────────────────────────────┘
                                │
┌───────────────────────────────┼──────────────────────────────────┐
│                           Server                                 │
│  ┌────────────────────────────┼──────────────────────────────┐  │
│  │              API Gateway (REST + WebSocket)               │  │
│  └────────────────────────────┼──────────────────────────────┘  │
│                               │                                  │
│  ┌──────────┐ ┌──────────┐  ┌┴─────────┐ ┌──────────┐ ┌──────┐ │
│  │   Auth   │ │   File   │  │  Chunk   │ │   Sync   │ │ Web- │ │
│  │ Service  │ │ Service  │  │ Service  │ │ Service  │ │ DAV  │ │
│  └────┬─────┘ └────┬─────┘  └────┬─────┘ └────┬─────┘ └──┬───┘ │
│       │            │             │            │            │     │
│  ┌────┴────┐ ┌────┴────┐ ┌──────┴────┐ ┌────┴────┐ ┌────┴───┐│
│  │  JWT    │ │   VFS   │ │    CDC    │ │  Vector │ │  Lock  ││
│  │  TOTP   │ │  Layer  │ │   Engine  │ │  Clocks │ │ Store  ││
│  └─────────┘ └─────────┘ └───────────┘ └─────────┘ └────────┘│
│                               │                                  │
│                    ┌──────────┴──────────┐                      │
│                    │   Storage Backend    │                      │
│                    │   (Local / S3)       │                      │
│                    └─────────────────────┘                      │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              SQLite (Embedded Database)                   │   │
│  │  Users │ Files │ Chunks │ Shares │ Versions │ Sync State │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration

Create a `config.yaml` file:

```yaml
server:
  host: 0.0.0.0
  port: 8443
  base_url: https://vault.example.com
  tls:
    enabled: true
    cert_file: /etc/vaultdrift/cert.pem
    key_file: /etc/vaultdrift/key.pem

storage:
  backend: local  # or s3
  local:
    data_dir: /var/lib/vaultdrift/data
  # s3:
  #   endpoint: s3.amazonaws.com
  #   bucket: my-bucket
  #   region: us-east-1
  #   access_key: ACCESS_KEY
  #   secret_key: SECRET_KEY

database:
  path: /var/lib/vaultdrift/vaultdrift.db

auth:
  jwt_secret: change-this-to-a-secure-random-string
  access_token_ttl: 15m
  refresh_token_ttl: 7d
  totp_enabled: true

sync:
  chunk_size_min: 262144      # 256KB
  chunk_size_avg: 1048576     # 1MB
  chunk_size_max: 4194304     # 4MB
  max_concurrent_transfers: 4

encryption:
  enabled: true
  zero_knowledge: true
  argon2_time: 3
  argon2_memory: 65536        # 64MB
  argon2_threads: 4

sharing:
  public_links_enabled: true
  max_expiry_days: 90
  default_expiry_days: 7

logging:
  level: info
  format: json
  audit: true
```

Environment variables override config values:
```bash
VAULTDRIFT_SERVER_PORT=8443
VAULTDRIFT_STORAGE_BACKEND=s3
VAULTDRIFT_STORAGE_S3_BUCKET=mybucket
VAULTDRIFT_AUTH_JWT_SECRET=secret
```

## Development

### Project Structure

```
vaultdrift/
├── cmd/
│   ├── server/              # Main server executable
│   ├── vaultdrift-cli/      # CLI client
│   └── vaultdrift-desktop/  # Desktop tray app
├── internal/
│   ├── api/                 # API types
│   ├── auth/                # Authentication & RBAC
│   ├── chunk/               # Content-defined chunking (CDC)
│   ├── cli/                 # CLI implementation
│   ├── config/              # Configuration
│   ├── crypto/              # Encryption/decryption
│   ├── db/                  # Database layer (SQLite)
│   ├── desktop/             # Desktop app
│   ├── server/              # HTTP server & middleware
│   ├── share/               # Sharing logic
│   ├── storage/             # Storage backends
│   ├── sync/                # Sync engine
│   ├── thumbnail/           # Thumbnail generation
│   ├── vfs/                 # Virtual file system
│   ├── webdav/              # WebDAV implementation
│   └── worker/              # Background workers
├── web/                     # React 19 + Tailwind CSS
│   ├── src/
│   │   ├── components/      # UI components
│   │   ├── lib/             # API client
│   │   ├── pages/           # Page components
│   │   └── stores/          # Zustand state
│   └── dist/                # Built assets (embedded)
├── docs/                    # Documentation
└── scripts/                 # Build & deploy scripts
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/chunk/... -v
go test ./internal/sync/... -v
```

### Building for Production

```bash
# Build all binaries for current platform
make build-all

# Cross-compile for multiple platforms
make build-cross

# Build Docker image
make docker-build

# Run with Docker Compose
make docker-up
```

## API Documentation

### Authentication

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user",
  "password": "pass",
  "totp_code": "123456"  # if 2FA enabled
}

Response:
{
  "token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": 1234567890
}
```

### Files

```http
# List files
GET /api/v1/files?parent_id={folder_id}
Authorization: Bearer {token}

# Create folder
POST /api/v1/folders
{
  "name": "My Folder",
  "parent_id": "uuid-or-null"
}

# Upload (chunked)
POST /api/v1/uploads
{
  "name": "file.pdf",
  "size_bytes": 10485760,
  "mime_type": "application/pdf",
  "parent_id": "folder-uuid"
}

# Download
GET /api/v1/downloads/{file_id}
```

### WebSocket (Real-time)

```javascript
const ws = new WebSocket('wss://vault.example.com/ws?token=JWT');

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  // Handle: file_created, file_updated, file_deleted, sync_required
};

// Subscribe to folder
ws.send(JSON.stringify({
  type: 'subscribe',
  payload: { folder_id: 'folder-uuid' }
}));
```

## Security

| Layer | Technology |
|-------|------------|
| Transport | TLS 1.3 with automatic Let's Encrypt |
| Passwords | Argon2id (OWASP recommended) |
| File Encryption | AES-256-GCM with unique keys per file |
| Key Exchange | X25519 ECDH for sharing |
| Tokens | JWT with Ed25519 signatures |
| 2FA | TOTP (RFC 6238) |
| Conflict Resolution | Vector clocks (Lamport timestamps) |

### Threat Model

- **Server compromise**: Files remain encrypted, only metadata exposed
- **Man-in-the-middle**: TLS 1.3 prevents interception
- **Password breach**: Argon2id makes offline cracking expensive
- **Replay attacks**: Short-lived JWT tokens with rotation
- **CSRF**: SameSite cookies and origin validation

## Deployment

### Docker

```bash
docker run -d \
  --name vaultdrift \
  -p 8443:8443 \
  -v /var/lib/vaultdrift:/data \
  -e VAULTDRIFT_AUTH_JWT_SECRET=secret \
  vaultdrift/vaultdrift:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  vaultdrift:
    image: vaultdrift/vaultdrift:latest
    ports:
      - "8443:8443"
    volumes:
      - ./data:/data
    environment:
      - VAULTDRIFT_STORAGE_BACKEND=s3
      - VAULTDRIFT_STORAGE_S3_BUCKET=mybucket
      - VAULTDRIFT_STORAGE_S3_ACCESS_KEY=${S3_KEY}
      - VAULTDRIFT_STORAGE_S3_SECRET_KEY=${S3_SECRET}
```

### Kubernetes

See `deploy/kubernetes/` for Helm charts and manifests.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Roadmap

- [ ] Mobile apps (iOS/Android)
- [ ] FUSE filesystem mount
- [ ] Office document collaboration
- [ ] Video streaming/transcoding
- [ ] Federation between servers
- [ ] IPFS backend support

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Rabin fingerprinting for content-defined chunking
- Vector clocks for distributed conflict resolution
- Merkle trees for efficient sync state comparison
- Argon2 password hashing competition

---

*VaultDrift — Zero dependencies. One binary. Complete control.*
