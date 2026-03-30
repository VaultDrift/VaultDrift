# VaultDrift — SPECIFICATION.md

## 🏷️ Project Identity

| Key | Value |
|-----|-------|
| **Name** | VaultDrift |
| **Tagline** | "Your Files. Your Vault. Your Drift." |
| **Domain** | vaultdrift.com |
| **GitHub** | github.com/vaultdrift/vaultdrift |
| **License** | Apache 2.0 |
| **Language** | Go 1.23+ |
| **Binary** | `vaultdrift` |
| **Philosophy** | #NOFORKANYMORE — Zero external dependencies, single binary, replaces Nextcloud/ownCloud/Seafile |

---

## 📋 Problem Statement

Self-hosted file sync & share is a mess:

| Solution | Pain Points |
|----------|-------------|
| **Nextcloud** | PHP + MySQL/PostgreSQL + Redis. Chronic performance issues. App ecosystem riddled with security holes. PHP dependency hell. Needs Apache/Nginx, cron, opcache tuning. |
| **ownCloud** | Legacy PHP version is dead. Infinite Scale (Go) still feels beta. Fragmented migration path. |
| **Seafile** | C + Python + MySQL + Memcached. Compiling from source is a nightmare. Two separate servers (seafile + seahub). |
| **Syncthing** | P2P only, no web UI for file management, no share links, no WebDAV. |
| **All of them** | Sync conflict resolution is broken. No single binary covers file sync + share + WebDAV + E2E encryption. |

**VaultDrift solves this**: One Go binary. Zero external dependencies. CobaltDB embedded. Local + S3 storage. WebDAV server built-in. Delta sync. E2E encryption. Public share links. Web UI + CLI + Desktop tray client.

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    VaultDrift Binary                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐ │
│  │ HTTP/API │  │  WebDAV  │  │   Sync   │  │  Admin  │ │
│  │  Server  │  │  Server  │  │ Protocol │  │   API   │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬────┘ │
│       │              │             │             │       │
│  ┌────┴──────────────┴─────────────┴─────────────┴────┐ │
│  │                  Core Engine                        │ │
│  │  ┌──────────┐ ┌──────────┐ ┌───────────────────┐  │ │
│  │  │   VFS    │ │  Crypto  │ │  Conflict Resolver │  │ │
│  │  │  Layer   │ │  Engine  │ │                    │  │ │
│  │  └────┬─────┘ └──────────┘ └───────────────────┘  │ │
│  └───────┼────────────────────────────────────────────┘ │
│          │                                              │
│  ┌───────┴────────────────────────────────────────────┐ │
│  │              Storage Abstraction Layer               │ │
│  │  ┌─────────────┐         ┌──────────────────────┐  │ │
│  │  │  Local FS   │         │  S3-Compatible       │  │ │
│  │  │  Backend    │         │  (AWS/MinIO/R2)      │  │ │
│  │  └─────────────┘         └──────────────────────┘  │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                         │
│  ┌─────────────────────────────────────────────────────┐ │
│  │              CobaltDB (Embedded)                     │ │
│  │  Metadata · Users · Sessions · Shares · Sync State  │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                         │
│  ┌─────────────────────────────────────────────────────┐ │
│  │           Embedded Web UI (SPA)                      │ │
│  │  Alpine.js + Tailwind CDN + File Manager             │ │
│  └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘

External Clients:
┌──────────┐  ┌───────────┐  ┌──────────────┐
│ CLI Sync │  │  Desktop  │  │   Mobile     │
│ Daemon   │  │  Tray App │  │  (Future)    │
└──────────┘  └───────────┘  └──────────────┘
```

---

## 🔧 Technical Decisions

### Language & Dependencies
- **Go 1.23+** — Single binary compilation, excellent concurrency
- **Zero external Go dependencies** — All functionality implemented from scratch or using Go stdlib
- **CobaltDB** — Embedded database engine (own project, dogfooding: B+Tree, WAL, MVCC, SQL parser, AES-256-GCM)
- **No CGo** — Pure Go, cross-compile everywhere

### Storage Architecture
- **Storage Abstraction Layer (SAL)** — Interface-based, pluggable backends
- **Local Filesystem Backend** — Direct OS file operations, inotify/fsevents for change detection
- **S3-Compatible Backend** — AWS S3, MinIO, Cloudflare R2, Backblaze B2 via custom S3 client (no aws-sdk-go)
- **Content-Addressable Chunks** — Files split into variable-size chunks (CDC — Content-Defined Chunking), stored by SHA-256 hash → automatic deduplication

### Chunking & Delta Sync
- **Content-Defined Chunking (CDC)** — Rabin fingerprint based, variable chunk sizes (min 256KB, avg 1MB, max 4MB)
- **Chunk-level deduplication** — Same chunk across files/users stored once
- **Delta sync** — Only changed chunks transferred (rsync-like efficiency without rsync dependency)
- **Chunk manifest** — Each file version = ordered list of chunk hashes + metadata

### Embedded Database (CobaltDB)
All metadata stored in CobaltDB:
- User accounts, sessions, RBAC roles & permissions
- File/folder tree (virtual filesystem metadata)
- Chunk index (hash → storage location mapping)
- Sync state (per-device vector clocks)
- Share links, audit log, encryption key metadata

### Encryption
- **E2E Encryption (Zero-Knowledge)** — Server never sees plaintext
- **Per-file encryption keys** — AES-256-GCM, random key per file
- **Key wrapping** — File keys encrypted with user's master key
- **Master key derivation** — Argon2id from user passphrase (not stored on server)
- **Encrypted chunk storage** — Chunks encrypted before storage, server handles opaque blobs
- **Key sharing** — File sharing = re-encrypting file key with recipient's public key (X25519)
- **Recovery key** — Optional 24-word mnemonic for master key recovery

### Sync Protocol
- **Vector clock per device** — Detects concurrent modifications
- **Merkle tree per folder** — Fast subtree comparison for sync negotiation
- **Three-phase sync**: 
  1. **Negotiate** — Client sends folder Merkle root, server responds with diff
  2. **Transfer** — Only changed chunks sent (bidirectional)
  3. **Commit** — Atomic metadata update after successful transfer
- **WebSocket for real-time** — Push notifications for changes
- **HTTP/2 for bulk transfer** — Multiplexed chunk upload/download

### Conflict Resolution
- **Vector clock comparison** — Detects true conflicts vs simple overwrites
- **Automatic resolution** for non-conflicting concurrent edits (different file regions)
- **Conflict files** for true conflicts: `document.txt` → `document.conflict-<device>-<timestamp>.txt`
- **Conflict dashboard** in Web UI — View, compare, merge, choose winner
- **Last-writer-wins option** — Configurable per folder for teams that prefer simplicity

---

## 👤 Multi-User & RBAC

### User Management
- **Local authentication** — Username/password, Argon2id hashing
- **TOTP 2FA** — Built-in TOTP support (Google Authenticator compatible)
- **API tokens** — Per-user, scoped, revocable tokens for CLI/API access
- **Session management** — JWT-based, refresh token rotation

### Role-Based Access Control (RBAC)

| Role | Capabilities |
|------|-------------|
| **Admin** | Full system control, user management, global settings, storage quotas |
| **User** | Own files CRUD, share (within permissions), sync, WebDAV access |
| **Guest** | Read/download shared files only (via share links or invitation) |
| **Custom** | Admin-definable roles with granular permissions |

### Permissions Model
```
Permission = {Resource, Action, Scope}

Resources: file, folder, share, user, system
Actions:   read, write, delete, share, manage
Scope:     own, group, all
```

### Quotas
- **Per-user storage quota** — Admin configurable
- **Per-user bandwidth quota** — Optional monthly transfer limit
- **Deduplication-aware** — Shared chunks don't double-count

---

## 🌐 API Design

### REST JSON API

Base: `POST/GET/PUT/DELETE /api/v1/*`

#### Authentication
```
POST   /api/v1/auth/login          → JWT token pair
POST   /api/v1/auth/refresh        → Refresh access token
POST   /api/v1/auth/logout         → Invalidate session
POST   /api/v1/auth/totp/setup     → Generate TOTP secret
POST   /api/v1/auth/totp/verify    → Verify TOTP code
```

#### Files & Folders
```
GET    /api/v1/fs/list              → List directory contents
GET    /api/v1/fs/info/{path}       → File/folder metadata
POST   /api/v1/fs/mkdir             → Create directory
PUT    /api/v1/fs/rename            → Rename/move file or folder
DELETE /api/v1/fs/delete            → Delete (soft delete → trash)
POST   /api/v1/fs/copy              → Copy file or folder
GET    /api/v1/fs/search            → Full-text filename search
GET    /api/v1/fs/recent            → Recently modified files
GET    /api/v1/fs/trash             → List trash contents
POST   /api/v1/fs/trash/restore     → Restore from trash
DELETE /api/v1/fs/trash/purge       → Permanent delete
```

#### Upload & Download
```
POST   /api/v1/upload/init          → Initialize chunked upload session
POST   /api/v1/upload/chunk         → Upload single chunk
POST   /api/v1/upload/complete      → Finalize upload (assemble manifest)
GET    /api/v1/download/{path}      → Download file (streaming)
GET    /api/v1/download/zip         → Download folder as ZIP (on-the-fly)
GET    /api/v1/thumbnail/{path}     → Image/video thumbnail (generated)
```

#### Sync
```
POST   /api/v1/sync/negotiate       → Send Merkle root, get diff
POST   /api/v1/sync/push            → Push changed chunks to server
GET    /api/v1/sync/pull             → Pull changed chunks from server
POST   /api/v1/sync/commit          → Commit sync transaction
GET    /api/v1/sync/status           → Current sync state
WS     /api/v1/sync/ws              → WebSocket for real-time change notifications
```

#### Sharing
```
POST   /api/v1/share/link           → Create public share link
GET    /api/v1/share/links          → List active share links
PUT    /api/v1/share/link/{id}      → Update share link settings
DELETE /api/v1/share/link/{id}      → Revoke share link
POST   /api/v1/share/user           → Share with specific user
GET    /api/v1/share/received       → Files shared with me
```

#### Users & Admin
```
GET    /api/v1/users                → List users (admin)
POST   /api/v1/users                → Create user (admin)
PUT    /api/v1/users/{id}           → Update user (admin)
DELETE /api/v1/users/{id}           → Delete user (admin)
GET    /api/v1/users/me             → Current user profile
PUT    /api/v1/users/me             → Update own profile
GET    /api/v1/admin/stats          → System statistics
PUT    /api/v1/admin/settings       → System settings
GET    /api/v1/admin/audit          → Audit log
```

### WebDAV Server
- **Endpoint**: `/dav/{username}/` 
- **Full WebDAV compliance**: PROPFIND, PROPPATCH, MKCOL, GET, PUT, DELETE, COPY, MOVE, LOCK, UNLOCK
- **Class 2 WebDAV** — Locking support for office document editing
- **Works with**: macOS Finder, Windows Explorer, Linux Nautilus/Dolphin, Cyberduck, rclone
- **Auth**: HTTP Basic over HTTPS (API token as password)
- **Encryption**: WebDAV operations transparently encrypt/decrypt if E2E is enabled

---

## 🖥️ Web UI (Embedded SPA)

### Technology
- **Alpine.js** — Reactive framework, minimal footprint
- **Tailwind CSS CDN** — Utility-first styling
- **Embedded in binary** — `go:embed` all static assets
- **No build step** — No node_modules, no webpack, no npm

### UI Modules

#### File Manager
- **Dual view**: Grid (thumbnails) / List (detailed)
- **Drag & drop** upload with progress bars
- **Multi-select** with bulk actions (move, delete, share, download ZIP)
- **Breadcrumb navigation** with quick folder jump
- **Context menu** (right-click) for file operations
- **Inline rename** with double-click
- **Preview panel**: Images, PDF, text, markdown, video, audio
- **Search bar** with instant results

#### Sync Dashboard
- **Device list** with last sync timestamp and status
- **Conflict queue** with side-by-side diff viewer
- **Sync activity log** — Real-time feed of sync events
- **Bandwidth monitor** — Upload/download rates

#### Share Manager
- **Active shares list** with stats (views, downloads)
- **Create share dialog**: Password, expiry, download limit, preview-only mode
- **QR code** generation for share links
- **Share with user**: Autocomplete user search, permission selector

#### Admin Panel
- **User management**: Create, edit, disable, quota management
- **Storage overview**: Usage by user, total capacity, dedup savings
- **System health**: CPU, memory, disk I/O, active connections
- **Audit log viewer**: Filterable by user, action, date
- **Settings**: Storage backend config, SMTP, branding

#### Profile & Settings
- **Profile**: Avatar, display name, email, password change
- **2FA setup**: QR code + backup codes
- **API tokens**: Generate, list, revoke
- **Encryption**: Master key setup, recovery key download
- **Connected devices**: List, rename, revoke

---

## 🖥️ CLI Client (Sync Daemon)

### Binary
- **Name**: `vaultdrift-cli`
- **Separate binary** — But same Go module, shared sync protocol code
- **Cross-platform**: Linux, macOS, Windows

### Commands
```
vaultdrift-cli init                     → Initialize sync folder, authenticate
vaultdrift-cli login                    → Authenticate with server
vaultdrift-cli sync                     → One-time sync
vaultdrift-cli daemon start             → Start background sync daemon
vaultdrift-cli daemon stop              → Stop daemon
vaultdrift-cli daemon status            → Show sync status
vaultdrift-cli ls [path]                → List remote files
vaultdrift-cli upload <local> <remote>  → Upload file
vaultdrift-cli download <remote> [local]→ Download file
vaultdrift-cli share <path>             → Create share link
vaultdrift-cli config set <key> <value> → Configure settings
vaultdrift-cli conflicts                → List unresolved conflicts
vaultdrift-cli conflicts resolve <id>   → Resolve a conflict
```

### Sync Daemon Behavior
- **Filesystem watcher**: inotify (Linux), FSEvents (macOS), ReadDirectoryChangesW (Windows)
- **Selective sync**: Choose which folders to sync locally
- **Bandwidth throttling**: Configurable upload/download limits
- **Pause/resume**: Manual control over sync
- **Retry with exponential backoff**: For network failures
- **Sync interval**: Configurable polling fallback (default: 30s) alongside real-time WebSocket

---

## 🖥️ Desktop Tray App

### Technology
- **Pure Go** — Using `systray` approach (no Electron, no web runtime)
- **Same binary** as CLI with `--tray` flag, or separate `vaultdrift-desktop` binary
- **Cross-platform**: Linux (AppIndicator), macOS (NSStatusBar), Windows (Shell_NotifyIcon)

### Tray Menu
```
VaultDrift
├── ✅ Synced (or ⟳ Syncing... or ⚠️ 2 conflicts)
├── Open Web UI → Opens browser to server
├── Open Sync Folder → Opens file manager
├── ─────────────
├── Recent Activity → Submenu with last 10 actions
├── Sync Now → Force immediate sync
├── Pause Sync / Resume Sync
├── ─────────────
├── Preferences → Opens settings window
├── ⚠️ Resolve Conflicts → Opens conflict resolver
├── ─────────────
└── Quit VaultDrift
```

### Desktop Notifications
- Sync complete
- New file shared with you
- Conflict detected
- Storage quota warning

---

## 🔐 Security Architecture

### Transport Security
- **TLS 1.3** built-in (Let's Encrypt auto-cert via ACME, or custom cert)
- **HTTP/2** by default
- **HSTS, CSP, X-Frame-Options** headers

### E2E Encryption Detail

```
Key Hierarchy:

User Passphrase
      │
      ▼ (Argon2id: time=3, memory=64MB, threads=4)
Master Key (256-bit)
      │
      ├──▶ File Key 1 (AES-256-GCM, random) ──▶ Encrypts File 1 chunks
      ├──▶ File Key 2 (AES-256-GCM, random) ──▶ Encrypts File 2 chunks
      └──▶ ...
      
Sharing:
User A                          User B
  │                                │
  Master Key A                     Master Key B
  │                                │
  X25519 Keypair A                 X25519 Keypair B
  │                                │
  └── File Key ──(encrypt with B's public key)──▶ User B can decrypt
```

- **Zero-knowledge**: Server stores only encrypted blobs and encrypted key bundles
- **Metadata encryption**: Filename, size visible to server (required for sync). Optional full metadata encryption mode (filenames encrypted, server sees only opaque paths)
- **Recovery key**: 24-word BIP39-style mnemonic that can reconstruct master key

### Server Security
- **Rate limiting** — Per-IP and per-user, token bucket algorithm
- **CSRF protection** — SameSite cookies + custom header validation
- **Input validation** — Path traversal prevention, filename sanitization
- **Audit logging** — All file operations, auth events, admin actions logged
- **Brute-force protection** — Progressive delays after failed login attempts

---

## 📁 Storage Layout

### Local Filesystem Backend
```
/var/lib/vaultdrift/
├── data/
│   ├── chunks/
│   │   ├── ab/                     ← First 2 chars of SHA-256
│   │   │   ├── ab3f8c...a1.chunk   ← Encrypted chunk blob
│   │   │   └── ab91d2...f3.chunk
│   │   ├── cd/
│   │   └── ...
│   └── thumbnails/
│       └── {user_id}/
│           └── {file_hash}.webp
├── db/
│   └── vaultdrift.cdb              ← CobaltDB database file
├── config/
│   └── vaultdrift.yaml             ← Server configuration
├── certs/
│   ├── cert.pem
│   └── key.pem
└── logs/
    └── vaultdrift.log
```

### S3 Backend Layout
```
s3://vaultdrift-bucket/
├── chunks/
│   ├── ab/ab3f8c...a1.chunk
│   └── ...
└── thumbnails/
    └── {user_id}/{file_hash}.webp
```

---

## ⚙️ Configuration

### Server Config (`vaultdrift.yaml`)
```yaml
server:
  host: 0.0.0.0
  port: 8443
  tls:
    enabled: true
    auto_cert: true                # Let's Encrypt ACME
    cert_file: ""                  # Or manual cert path
    key_file: ""
  base_url: "https://vault.example.com"

storage:
  backend: local                   # local | s3
  local:
    data_dir: /var/lib/vaultdrift/data
  s3:
    endpoint: ""                   # MinIO/R2 endpoint
    bucket: ""
    region: ""
    access_key: ""
    secret_key: ""
    use_path_style: false          # true for MinIO

database:
  path: /var/lib/vaultdrift/db/vaultdrift.cdb

auth:
  jwt_secret: ""                   # Auto-generated if empty
  access_token_ttl: 15m
  refresh_token_ttl: 7d
  totp_enabled: true
  max_login_attempts: 5
  lockout_duration: 15m

sync:
  chunk_size_min: 262144           # 256KB
  chunk_size_avg: 1048576          # 1MB
  chunk_size_max: 4194304          # 4MB
  max_concurrent_transfers: 4
  websocket_enabled: true

encryption:
  enabled: true
  zero_knowledge: true             # E2E mode
  argon2_time: 3
  argon2_memory: 65536             # 64MB
  argon2_threads: 4

sharing:
  public_links_enabled: true
  max_expiry_days: 90
  default_expiry_days: 7
  password_required: false         # Force password on all shares
  max_download_limit: 1000

users:
  registration_enabled: false      # Admin-only user creation
  default_quota: 10GB
  max_quota: 0                     # 0 = unlimited

smtp:
  enabled: false
  host: ""
  port: 587
  username: ""
  password: ""
  from: "noreply@vault.example.com"

logging:
  level: info                      # debug, info, warn, error
  file: /var/lib/vaultdrift/logs/vaultdrift.log
  audit: true
```

---

## 📦 Deliverables

### Binaries
| Binary | Description |
|--------|-------------|
| `vaultdrift` | Server binary (HTTP API + WebDAV + Web UI + Sync server) |
| `vaultdrift-cli` | CLI client (sync daemon + file operations) |
| `vaultdrift-desktop` | Desktop tray app (GUI sync client) |

### Installation Methods
```bash
# Single binary download
curl -fsSL https://vaultdrift.com/install.sh | sh

# Docker (single container)
docker run -d -p 8443:8443 -v vaultdrift-data:/var/lib/vaultdrift vaultdrift/vaultdrift

# From source
git clone https://github.com/vaultdrift/vaultdrift
cd vaultdrift && go build ./cmd/vaultdrift
```

### First Run
```bash
# Initialize with admin user
vaultdrift init --admin-user admin --admin-email admin@example.com

# Start server
vaultdrift serve

# Or with config file
vaultdrift serve --config /etc/vaultdrift/vaultdrift.yaml
```

---

## 📊 Performance Targets

| Metric | Target |
|--------|--------|
| File listing (10K files) | < 100ms |
| Single file upload (100MB) | Wire speed (limited by network) |
| Chunked upload overhead | < 5% vs raw transfer |
| Delta sync (1% changed) | Transfer only changed chunks (~1% of file) |
| WebDAV PROPFIND | < 50ms for 1K entries |
| Concurrent users | 100+ on modest hardware (2 CPU, 4GB RAM) |
| Memory per connection | < 2MB |
| Cold start to serving | < 500ms |
| Dedup ratio (typical) | 30-60% storage savings |
| Thumbnail generation | < 200ms per image |

---

## 🗺️ Roadmap

### Phase 1 — Core (MVP)
- [x] Storage abstraction (local + S3)
- [x] CobaltDB integration
- [x] Content-defined chunking + dedup
- [x] REST API (files, folders, upload, download)
- [x] User auth (JWT, Argon2id, TOTP)
- [x] RBAC (admin, user, guest, custom)
- [x] WebDAV server (Class 2)
- [x] Delta sync protocol (Merkle tree + vector clocks)
- [x] E2E encryption (AES-256-GCM, X25519, Argon2id)
- [x] Public share links (password, expiry, download limit)
- [x] Web UI (Alpine.js + Tailwind, file manager, share manager)
- [x] CLI client (sync daemon, file operations)
- [x] Conflict detection & resolution

### Phase 2 — Polish
- [ ] Desktop tray app (Linux, macOS, Windows)
- [ ] Image/video thumbnails
- [ ] File versioning (configurable retention)
- [ ] Trash with auto-purge
- [ ] Full-text search (content indexing)
- [ ] Activity feed & notifications
- [ ] SMTP integration (share invites, alerts)
- [ ] Bandwidth throttling

### Phase 3 — Advanced
- [ ] Collaborative editing (CRDT-based for text files)
- [ ] Office document preview (LibreOffice integration optional)
- [ ] S3 lifecycle policies (tiered storage)
- [ ] Multi-node clustering (Raft consensus for metadata)
- [ ] LDAP/OIDC authentication
- [ ] Mobile clients (iOS, Android)
- [ ] Plugin system for extensibility
- [ ] Federation (server-to-server sharing)

---

## 🏷️ Competitive Positioning

| Feature | VaultDrift | Nextcloud | ownCloud IS | Seafile | Syncthing |
|---------|-----------|-----------|-------------|---------|-----------|
| Single binary | ✅ | ❌ (PHP stack) | ⚠️ (multiple services) | ❌ (C+Python) | ✅ |
| Zero deps | ✅ | ❌ | ❌ | ❌ | ✅ |
| Web UI | ✅ | ✅ | ✅ | ✅ | ❌ |
| WebDAV | ✅ | ✅ | ✅ | ⚠️ (via gateway) | ❌ |
| Delta sync | ✅ (CDC) | ❌ | ⚠️ | ✅ | ✅ |
| E2E encryption | ✅ (built-in) | ⚠️ (plugin) | ❌ | ✅ | ✅ (transport) |
| Share links | ✅ | ✅ | ✅ | ✅ | ❌ |
| S3 backend | ✅ | ✅ (plugin) | ✅ | ❌ | ❌ |
| Deduplication | ✅ (chunk-level) | ❌ | ❌ | ✅ (block) | ❌ |
| Conflict resolution | ✅ (vector clock) | ⚠️ (basic) | ⚠️ | ⚠️ | ✅ (vector clock) |
| Memory usage | Low (~50MB) | High (~500MB+) | Medium (~200MB) | Medium | Low |
| Setup time | < 1 min | 30+ min | 15+ min | 20+ min | < 1 min |

---

## 🧪 Testing Strategy

| Layer | Approach |
|-------|----------|
| **Unit tests** | Every package, 80%+ coverage |
| **Integration tests** | Full API test suite, WebDAV compliance tests |
| **Sync tests** | Multi-client conflict scenarios, network partition simulation |
| **Encryption tests** | Known-answer tests, key derivation vectors |
| **Benchmark tests** | Chunk throughput, sync negotiation latency, concurrent upload |
| **Fuzz tests** | API input fuzzing, WebDAV request fuzzing, chunk parsing |
| **E2E tests** | CLI ↔ Server ↔ Web UI full workflow tests |

---

*VaultDrift — Your Files. Your Vault. Your Drift.*
*Zero dependencies. One binary. Complete control.*
