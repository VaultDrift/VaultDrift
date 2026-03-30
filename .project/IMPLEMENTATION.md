# VaultDrift — IMPLEMENTATION.md

## 📁 Project Structure

```
vaultdrift/
├── cmd/
│   ├── vaultdrift/                 # Server binary
│   │   └── main.go
│   ├── vaultdrift-cli/             # CLI client binary
│   │   └── main.go
│   └── vaultdrift-desktop/         # Desktop tray binary
│       └── main.go
├── internal/
│   ├── server/                     # HTTP server, router, middleware
│   │   ├── server.go               # Main server struct, lifecycle
│   │   ├── router.go               # Route registration
│   │   ├── middleware.go            # Auth, CORS, rate limit, logging
│   │   └── context.go              # Request context helpers
│   ├── api/                        # REST API handlers
│   │   ├── auth.go                 # Login, logout, refresh, TOTP
│   │   ├── files.go                # List, info, mkdir, rename, delete, copy
│   │   ├── upload.go               # Chunked upload init, chunk, complete
│   │   ├── download.go             # File download, ZIP, thumbnails
│   │   ├── sync.go                 # Sync negotiate, push, pull, commit
│   │   ├── share.go                # Share links, user sharing
│   │   ├── users.go                # User CRUD, profile
│   │   ├── admin.go                # Admin stats, settings, audit
│   │   └── websocket.go            # WebSocket upgrade, change feed
│   ├── webdav/                     # WebDAV server implementation
│   │   ├── server.go               # WebDAV handler, method dispatch
│   │   ├── propfind.go             # PROPFIND response builder
│   │   ├── proppatch.go            # PROPPATCH handler
│   │   ├── lock.go                 # LOCK/UNLOCK (Class 2)
│   │   ├── methods.go              # GET, PUT, DELETE, MKCOL, COPY, MOVE
│   │   └── xml.go                  # WebDAV XML marshaling/unmarshaling
│   ├── vfs/                        # Virtual Filesystem layer
│   │   ├── vfs.go                  # VFS interface + core operations
│   │   ├── tree.go                 # In-memory file/folder tree
│   │   ├── node.go                 # FileNode, FolderNode types
│   │   ├── path.go                 # Path normalization, validation
│   │   ├── trash.go                # Soft delete, trash management
│   │   └── search.go               # Filename search, recent files
│   ├── storage/                    # Storage Abstraction Layer (SAL)
│   │   ├── storage.go              # Backend interface definition
│   │   ├── local.go                # Local filesystem backend
│   │   ├── s3.go                   # S3-compatible backend
│   │   └── s3client/               # Custom S3 client (no aws-sdk)
│   │       ├── client.go           # HTTP client, signing
│   │       ├── sign.go             # AWS Signature V4
│   │       ├── operations.go       # PutObject, GetObject, DeleteObject, ListObjects
│   │       └── multipart.go        # Multipart upload for large chunks
│   ├── chunk/                      # Content-Defined Chunking engine
│   │   ├── chunker.go              # CDC implementation (Rabin fingerprint)
│   │   ├── rabin.go                # Rabin hash rolling window
│   │   ├── manifest.go             # File manifest (ordered chunk list)
│   │   ├── dedup.go                # Deduplication logic
│   │   └── reassemble.go           # Chunk reassembly into file stream
│   ├── crypto/                     # Encryption engine
│   │   ├── crypto.go               # High-level encrypt/decrypt operations
│   │   ├── aes.go                  # AES-256-GCM encrypt/decrypt
│   │   ├── argon2.go               # Argon2id key derivation
│   │   ├── x25519.go               # X25519 key exchange
│   │   ├── keys.go                 # Master key, file key, key wrapping
│   │   ├── recovery.go             # BIP39-style recovery key (24 words)
│   │   └── random.go               # Cryptographic random helpers
│   ├── sync/                       # Sync protocol engine
│   │   ├── engine.go               # Sync orchestrator
│   │   ├── merkle.go               # Merkle tree construction & comparison
│   │   ├── vector.go               # Vector clock implementation
│   │   ├── diff.go                 # Diff calculation from Merkle comparison
│   │   ├── transfer.go             # Chunk transfer manager
│   │   ├── conflict.go             # Conflict detection & resolution
│   │   └── watcher.go              # Filesystem change watcher (server-side)
│   ├── auth/                       # Authentication & authorization
│   │   ├── auth.go                 # Auth service, login flow
│   │   ├── jwt.go                  # JWT generation, validation, refresh
│   │   ├── totp.go                 # TOTP implementation (RFC 6238)
│   │   ├── password.go             # Argon2id password hashing
│   │   ├── rbac.go                 # Role-based access control
│   │   ├── session.go              # Session management
│   │   └── token.go                # API token management
│   ├── share/                      # Sharing engine
│   │   ├── share.go                # Share service
│   │   ├── link.go                 # Public link generation, validation
│   │   ├── access.go               # Share access control
│   │   └── qrcode.go               # QR code generation (pure Go)
│   ├── db/                         # CobaltDB integration layer
│   │   ├── db.go                   # Database manager, migrations
│   │   ├── schema.go               # Table definitions, indexes
│   │   ├── users.go                # User queries
│   │   ├── files.go                # File metadata queries
│   │   ├── chunks.go               # Chunk index queries
│   │   ├── shares.go               # Share queries
│   │   ├── sessions.go             # Session queries
│   │   ├── syncstate.go            # Sync state queries
│   │   └── audit.go                # Audit log queries
│   ├── thumbnail/                  # Thumbnail generation
│   │   ├── thumbnail.go            # Thumbnail service
│   │   ├── image.go                # Image resize (pure Go)
│   │   └── cache.go                # Thumbnail cache management
│   ├── notify/                     # Notification system
│   │   ├── notify.go               # Notification dispatcher
│   │   ├── hub.go                  # WebSocket hub (fan-out)
│   │   └── smtp.go                 # Email notifications
│   ├── config/                     # Configuration management
│   │   ├── config.go               # Config struct, defaults
│   │   ├── loader.go               # YAML loader, env override
│   │   └── validate.go             # Config validation
│   ├── tls/                        # TLS & ACME
│   │   ├── tls.go                  # TLS config builder
│   │   └── acme.go                 # Let's Encrypt ACME client
│   └── util/                       # Shared utilities
│       ├── hash.go                 # SHA-256 helpers
│       ├── encoding.go             # Base64, hex encoding
│       ├── sanitize.go             # Path & filename sanitization
│       ├── size.go                 # Human-readable size formatting
│       ├── time.go                 # Time formatting, RFC helpers
│       └── pool.go                 # Byte buffer pool (sync.Pool)
├── client/                         # Shared client library
│   ├── client.go                   # API client
│   ├── sync.go                     # Client-side sync engine
│   ├── watcher/                    # Filesystem watcher
│   │   ├── watcher.go              # Platform abstraction
│   │   ├── watcher_linux.go        # inotify
│   │   ├── watcher_darwin.go       # FSEvents
│   │   └── watcher_windows.go      # ReadDirectoryChangesW
│   ├── daemon.go                   # Background sync daemon
│   ├── config.go                   # Client configuration
│   └── conflict.go                 # Client-side conflict handling
├── desktop/                        # Desktop tray app
│   ├── tray.go                     # System tray integration
│   ├── tray_linux.go               # AppIndicator / D-Bus StatusNotifier
│   ├── tray_darwin.go              # NSStatusBar via CGo-free approach
│   ├── tray_windows.go             # Shell_NotifyIcon
│   ├── menu.go                     # Tray menu builder
│   └── notification.go             # Desktop notifications
├── web/                            # Embedded Web UI
│   ├── embed.go                    # go:embed directive
│   ├── static/
│   │   ├── index.html              # SPA entry point
│   │   ├── app.js                  # Alpine.js application
│   │   ├── components/
│   │   │   ├── file-manager.js     # File browser component
│   │   │   ├── upload.js           # Upload manager component
│   │   │   ├── share-dialog.js     # Share creation dialog
│   │   │   ├── sync-dashboard.js   # Sync status dashboard
│   │   │   ├── admin-panel.js      # Admin panel component
│   │   │   ├── conflict-viewer.js  # Conflict resolution UI
│   │   │   ├── preview.js          # File preview panel
│   │   │   ├── auth.js             # Login, 2FA, registration
│   │   │   └── settings.js         # User settings, profile
│   │   ├── styles/
│   │   │   └── app.css             # Custom styles (minimal, Tailwind handles most)
│   │   └── icons/
│   │       └── ...                 # SVG icons embedded
│   └── handler.go                  # Static file serving handler
├── cobaltdb/                       # CobaltDB embedded (git subtree or module)
│   └── ...                         # Full CobaltDB engine
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── vaultdrift.yaml.example
├── README.md
├── LICENSE
├── SPECIFICATION.md
├── IMPLEMENTATION.md
├── TASKS.md
└── BRANDING.md
```

---

## 📊 Data Models

### CobaltDB Schema

#### Users Table
```sql
CREATE TABLE users (
    id              TEXT PRIMARY KEY,        -- UUID v7
    username        TEXT UNIQUE NOT NULL,
    email           TEXT UNIQUE NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    password_hash   TEXT NOT NULL,           -- Argon2id hash
    role            TEXT NOT NULL DEFAULT 'user', -- admin, user, guest, custom role name
    quota_bytes     INTEGER NOT NULL DEFAULT 10737418240, -- 10GB default
    used_bytes      INTEGER NOT NULL DEFAULT 0,
    totp_secret     TEXT DEFAULT NULL,       -- Encrypted TOTP secret
    totp_enabled    INTEGER NOT NULL DEFAULT 0,
    public_key      BLOB DEFAULT NULL,       -- X25519 public key (for E2E sharing)
    encrypted_private_key BLOB DEFAULT NULL, -- X25519 private key (encrypted with master key)
    recovery_key_hash TEXT DEFAULT NULL,     -- Hash of recovery key (verification only)
    avatar_chunk_hash TEXT DEFAULT NULL,     -- Reference to avatar image chunk
    status          TEXT NOT NULL DEFAULT 'active', -- active, disabled, locked
    last_login_at   TEXT DEFAULT NULL,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
```

#### Sessions Table
```sql
CREATE TABLE sessions (
    id              TEXT PRIMARY KEY,        -- UUID v7
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token   TEXT UNIQUE NOT NULL,    -- Hashed refresh token
    device_name     TEXT NOT NULL DEFAULT 'Unknown',
    device_type     TEXT NOT NULL DEFAULT 'web', -- web, cli, desktop, mobile
    ip_address      TEXT NOT NULL,
    user_agent      TEXT NOT NULL DEFAULT '',
    last_active_at  TEXT NOT NULL,
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_refresh ON sessions(refresh_token);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
```

#### API Tokens Table
```sql
CREATE TABLE api_tokens (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    token_hash      TEXT UNIQUE NOT NULL,    -- SHA-256 of token
    permissions     TEXT NOT NULL DEFAULT '[]', -- JSON array of permission strings
    last_used_at    TEXT DEFAULT NULL,
    expires_at      TEXT DEFAULT NULL,       -- NULL = never expires
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_hash ON api_tokens(token_hash);
```

#### Files Table (Virtual Filesystem)
```sql
CREATE TABLE files (
    id              TEXT PRIMARY KEY,        -- UUID v7
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id       TEXT DEFAULT NULL REFERENCES files(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,           -- Filename or folder name
    name_encrypted  BLOB DEFAULT NULL,       -- Encrypted filename (zero-knowledge mode)
    type            TEXT NOT NULL,           -- 'file' or 'folder'
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    mime_type       TEXT NOT NULL DEFAULT 'application/octet-stream',
    manifest_id     TEXT DEFAULT NULL REFERENCES manifests(id), -- Current version manifest
    checksum        TEXT DEFAULT NULL,       -- SHA-256 of complete file
    is_encrypted    INTEGER NOT NULL DEFAULT 0,
    encrypted_key   BLOB DEFAULT NULL,       -- File encryption key (wrapped with user's master key)
    is_trashed      INTEGER NOT NULL DEFAULT 0,
    trashed_at      TEXT DEFAULT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    
    UNIQUE(user_id, parent_id, name)        -- No duplicate names in same folder
);

CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_parent ON files(parent_id);
CREATE INDEX idx_files_user_parent ON files(user_id, parent_id);
CREATE INDEX idx_files_name ON files(name);
CREATE INDEX idx_files_type ON files(type);
CREATE INDEX idx_files_trashed ON files(is_trashed);
CREATE INDEX idx_files_updated ON files(updated_at);
CREATE INDEX idx_files_mime ON files(mime_type);
```

#### Manifests Table (File Versions)
```sql
CREATE TABLE manifests (
    id              TEXT PRIMARY KEY,        -- UUID v7
    file_id         TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    size_bytes      INTEGER NOT NULL,
    chunk_count     INTEGER NOT NULL,
    chunks          TEXT NOT NULL,           -- JSON: ordered array of chunk hashes
    checksum        TEXT NOT NULL,           -- SHA-256 of assembled file
    device_id       TEXT NOT NULL,           -- Which device created this version
    created_at      TEXT NOT NULL,

    UNIQUE(file_id, version)
);

CREATE INDEX idx_manifests_file ON manifests(file_id);
CREATE INDEX idx_manifests_file_version ON manifests(file_id, version);
```

#### Chunks Table
```sql
CREATE TABLE chunks (
    hash            TEXT PRIMARY KEY,        -- SHA-256 of chunk content (before encryption)
    size_bytes      INTEGER NOT NULL,
    storage_backend TEXT NOT NULL,           -- 'local' or 's3'
    storage_path    TEXT NOT NULL,           -- Path/key in backend
    ref_count       INTEGER NOT NULL DEFAULT 1, -- Reference counting for GC
    is_encrypted    INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_chunks_backend ON chunks(storage_backend);
CREATE INDEX idx_chunks_refcount ON chunks(ref_count);
```

#### Shares Table
```sql
CREATE TABLE shares (
    id              TEXT PRIMARY KEY,        -- UUID v7
    file_id         TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_by      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_type      TEXT NOT NULL,           -- 'link' or 'user'
    
    -- Link share fields
    token           TEXT UNIQUE DEFAULT NULL, -- URL-safe random token
    password_hash   TEXT DEFAULT NULL,       -- Argon2id hash (optional)
    expires_at      TEXT DEFAULT NULL,
    max_downloads   INTEGER DEFAULT NULL,    -- NULL = unlimited
    download_count  INTEGER NOT NULL DEFAULT 0,
    allow_upload    INTEGER NOT NULL DEFAULT 0,
    preview_only    INTEGER NOT NULL DEFAULT 0, -- View but not download
    
    -- User share fields
    shared_with     TEXT DEFAULT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission      TEXT NOT NULL DEFAULT 'read', -- read, write, manage
    encrypted_key   BLOB DEFAULT NULL,       -- File key re-encrypted for recipient
    
    is_active       INTEGER NOT NULL DEFAULT 1,
    view_count      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX idx_shares_file ON shares(file_id);
CREATE INDEX idx_shares_token ON shares(token);
CREATE INDEX idx_shares_created_by ON shares(created_by);
CREATE INDEX idx_shares_shared_with ON shares(shared_with);
CREATE INDEX idx_shares_active ON shares(is_active);
```

#### Devices Table (Sync)
```sql
CREATE TABLE devices (
    id              TEXT PRIMARY KEY,        -- UUID v7
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    device_type     TEXT NOT NULL,           -- 'cli', 'desktop', 'web', 'mobile'
    os              TEXT NOT NULL DEFAULT '',
    sync_folder     TEXT NOT NULL DEFAULT '', -- Local sync path on device
    last_sync_at    TEXT DEFAULT NULL,
    vector_clock    TEXT NOT NULL DEFAULT '{}', -- JSON: device_id → counter
    merkle_root     TEXT DEFAULT NULL,       -- Last known Merkle root hash
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX idx_devices_user ON devices(user_id);
CREATE INDEX idx_devices_active ON devices(is_active);
```

#### Sync State Table
```sql
CREATE TABLE sync_state (
    id              TEXT PRIMARY KEY,
    device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    file_id         TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    manifest_id     TEXT NOT NULL REFERENCES manifests(id),
    vector_clock    TEXT NOT NULL,           -- JSON vector clock at sync time
    synced_at       TEXT NOT NULL,

    UNIQUE(device_id, file_id)
);

CREATE INDEX idx_sync_state_device ON sync_state(device_id);
CREATE INDEX idx_sync_state_file ON sync_state(file_id);
```

#### RBAC Tables
```sql
CREATE TABLE roles (
    id              TEXT PRIMARY KEY,
    name            TEXT UNIQUE NOT NULL,    -- admin, user, guest, or custom
    description     TEXT NOT NULL DEFAULT '',
    is_system       INTEGER NOT NULL DEFAULT 0, -- System roles can't be deleted
    created_at      TEXT NOT NULL
);

CREATE TABLE permissions (
    id              TEXT PRIMARY KEY,
    role_id         TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource        TEXT NOT NULL,           -- file, folder, share, user, system
    action          TEXT NOT NULL,           -- read, write, delete, share, manage
    scope           TEXT NOT NULL DEFAULT 'own', -- own, group, all
    
    UNIQUE(role_id, resource, action, scope)
);

CREATE INDEX idx_permissions_role ON permissions(role_id);

CREATE TABLE user_roles (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id         TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    
    PRIMARY KEY(user_id, role_id)
);
```

#### Audit Log Table
```sql
CREATE TABLE audit_log (
    id              TEXT PRIMARY KEY,        -- UUID v7
    user_id         TEXT DEFAULT NULL,       -- NULL for system events
    action          TEXT NOT NULL,           -- file.upload, file.delete, auth.login, share.create, etc.
    resource_type   TEXT NOT NULL,           -- file, folder, user, share, system
    resource_id     TEXT DEFAULT NULL,
    details         TEXT NOT NULL DEFAULT '{}', -- JSON: additional context
    ip_address      TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_audit_user ON audit_log(user_id);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_resource ON audit_log(resource_type, resource_id);
CREATE INDEX idx_audit_created ON audit_log(created_at);
```

#### Settings Table
```sql
CREATE TABLE settings (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);
```

---

## 🔧 Core Module Specifications

### Module 1: Storage Abstraction Layer (`internal/storage/`)

#### Interface
```go
// Backend is the storage abstraction interface.
// All implementations must be safe for concurrent use.
type Backend interface {
    // Put stores a chunk blob. Key is the content hash.
    Put(ctx context.Context, key string, data []byte) error
    
    // Get retrieves a chunk blob by key.
    Get(ctx context.Context, key string) ([]byte, error)
    
    // Delete removes a chunk blob.
    Delete(ctx context.Context, key string) error
    
    // Exists checks if a chunk exists.
    Exists(ctx context.Context, key string) (bool, error)
    
    // List returns chunk keys with given prefix.
    List(ctx context.Context, prefix string) ([]string, error)
    
    // Stats returns storage usage statistics.
    Stats(ctx context.Context) (*StorageStats, error)
}

type StorageStats struct {
    TotalBytes   int64
    UsedBytes    int64
    ChunkCount   int64
    BackendType  string
}
```

#### Local Backend Implementation
- Key → path mapping: `hash[:2]/hash[2:].chunk` (2-char prefix directory for fanout)
- Atomic writes: Write to temp file → `os.Rename()` (atomic on same filesystem)
- Read: Direct `os.ReadFile()` with mmap for large chunks
- Delete: `os.Remove()` + remove empty parent dir
- Concurrent-safe: Each chunk is a separate file, no locking needed
- Stats: Walk data directory, sum sizes

#### S3 Backend Implementation
- Custom S3 client (no aws-sdk-go dependency)
- AWS Signature V4 signing from scratch (`internal/storage/s3client/sign.go`)
- Key format: `chunks/{hash[:2]}/{hash[2:]}.chunk`
- Multipart upload for chunks > 5MB
- Connection pooling via `http.Transport`
- Retry with exponential backoff (3 retries, 1s/2s/4s)
- Compatible with: AWS S3, MinIO, Cloudflare R2, Backblaze B2, DigitalOcean Spaces

### Module 2: Content-Defined Chunking (`internal/chunk/`)

#### Rabin Fingerprint CDC Algorithm
```
Parameters:
  - Window size: 48 bytes
  - Min chunk: 256KB (262,144 bytes)
  - Avg chunk: 1MB (1,048,576 bytes)  
  - Max chunk: 4MB (4,194,304 bytes)
  - Mask for avg: 0x000FFFFF (20 bits → ~1MB average)

Algorithm:
  1. Slide Rabin window over input stream
  2. At each byte, compute rolling hash
  3. If hash & mask == mask → chunk boundary
  4. Enforce min/max: skip boundary if < min, force boundary if > max
  5. SHA-256 hash each chunk
  6. Output: ordered list of {hash, offset, size}
```

#### Chunker Interface
```go
type Chunker struct {
    minSize   int  // 256KB
    avgSize   int  // 1MB  
    maxSize   int  // 4MB
    window    int  // 48 bytes
    mask      uint64
}

type ChunkInfo struct {
    Hash   string // SHA-256 hex
    Offset int64
    Size   int
}

// Chunk splits a reader into content-defined chunks.
// Returns manifest of chunk hashes in order.
func (c *Chunker) Chunk(r io.Reader) ([]ChunkInfo, error)

// ChunkEncrypted chunks and encrypts each chunk with the given key.
func (c *Chunker) ChunkEncrypted(r io.Reader, key []byte) ([]ChunkInfo, error)
```

#### Manifest Structure
```go
type Manifest struct {
    ID        string      // UUID v7
    FileID    string      // Parent file ID
    Version   int
    Size      int64       // Total file size
    Chunks    []string    // Ordered chunk hashes
    Checksum  string      // SHA-256 of complete file
    DeviceID  string
    CreatedAt time.Time
}
```

#### Reassembly
```go
// Reassemble streams chunks in order from storage to writer.
func Reassemble(ctx context.Context, manifest *Manifest, store storage.Backend, w io.Writer) error
// ReassembleDecrypt streams and decrypts chunks.
func ReassembleDecrypt(ctx context.Context, manifest *Manifest, store storage.Backend, key []byte, w io.Writer) error
```

### Module 3: Encryption Engine (`internal/crypto/`)

#### Key Derivation Flow
```
UserPassphrase (string)
    │
    ▼
Argon2id(passphrase, salt, time=3, mem=64MB, threads=4, keyLen=32)
    │
    ▼
MasterKey (32 bytes)
    │
    ├──▶ Stored nowhere on server
    ├──▶ Used to wrap/unwrap file keys (AES-256-GCM key-wrap)
    ├──▶ Used to encrypt/decrypt X25519 private key
    └──▶ Recovery: BIP39 mnemonic → same master key
```

#### Key Operations
```go
// DeriveKey derives master key from passphrase using Argon2id.
func DeriveKey(passphrase string, salt []byte) (masterKey [32]byte)

// GenerateFileKey creates a random AES-256 key for a file.
func GenerateFileKey() ([32]byte, error)

// WrapKey encrypts a file key with the master key (AES-256-GCM).
func WrapKey(fileKey, masterKey [32]byte) ([]byte, error)

// UnwrapKey decrypts a file key with the master key.
func UnwrapKey(wrappedKey []byte, masterKey [32]byte) ([32]byte, error)

// EncryptChunk encrypts chunk data with file key (AES-256-GCM).
// Returns: nonce (12 bytes) + ciphertext + tag (16 bytes)
func EncryptChunk(plaintext, fileKey []byte) ([]byte, error)

// DecryptChunk decrypts chunk data with file key.
func DecryptChunk(ciphertext, fileKey []byte) ([]byte, error)

// GenerateKeyPair creates X25519 keypair for E2E sharing.
func GenerateKeyPair() (publicKey, privateKey [32]byte, error)

// ShareFileKey re-encrypts file key for recipient using X25519.
func ShareFileKey(fileKey, senderPrivate, recipientPublic [32]byte) ([]byte, error)

// ReceiveFileKey decrypts shared file key using own private key.
func ReceiveFileKey(encryptedKey []byte, recipientPrivate, senderPublic [32]byte) ([32]byte, error)

// GenerateRecoveryKey creates 24-word BIP39 mnemonic for master key backup.
func GenerateRecoveryKey(masterKey [32]byte) (string, error)

// RecoverMasterKey restores master key from 24-word mnemonic.
func RecoverMasterKey(mnemonic string) ([32]byte, error)
```

### Module 4: Sync Engine (`internal/sync/`)

#### Merkle Tree
```go
// MerkleTree represents a hash tree over a directory structure.
type MerkleTree struct {
    Root     *MerkleNode
    NodeMap  map[string]*MerkleNode // path → node
}

type MerkleNode struct {
    Path     string
    Hash     string        // SHA-256 of concatenated children hashes (folder) or file checksum
    IsDir    bool
    Children []*MerkleNode // Sorted by name
    ModTime  time.Time
    Size     int64
}

// Build constructs Merkle tree from VFS for a user.
func Build(ctx context.Context, userID string, db *db.Manager) (*MerkleTree, error)

// Diff compares two Merkle trees and returns changed paths.
func Diff(local, remote *MerkleTree) (*DiffResult, error)

type DiffResult struct {
    Added    []string  // Paths only in remote
    Modified []string  // Paths in both but different hash
    Deleted  []string  // Paths only in local
}
```

#### Vector Clock
```go
// VectorClock tracks causality across devices.
type VectorClock map[string]uint64 // device_id → counter

// Increment bumps the counter for a device.
func (vc VectorClock) Increment(deviceID string)

// Merge combines two vector clocks (element-wise max).
func (vc VectorClock) Merge(other VectorClock)

// Compare returns the causal relationship.
func (vc VectorClock) Compare(other VectorClock) Ordering

type Ordering int
const (
    Before     Ordering = iota // vc happened before other
    After                      // vc happened after other
    Concurrent                 // True conflict
    Equal                      // Same state
)
```

#### Sync Protocol Flow
```
Phase 1: NEGOTIATE
───────────────────
Client                              Server
  │                                    │
  │── POST /sync/negotiate ──────────▶│
  │   { device_id, merkle_root }      │
  │                                    │
  │   Server compares Merkle roots     │
  │   If roots match → already synced  │
  │   If differ → compute subtree diff │
  │                                    │
  │◀── Response ──────────────────────│
  │   { diff: {added, modified,        │
  │     deleted}, server_clock }       │
  │                                    │

Phase 2: TRANSFER
───────────────────
Client                              Server
  │                                    │
  │── POST /sync/push ───────────────▶│
  │   { chunks: [{hash, data}...] }   │
  │   (Client uploads new/modified     │
  │    chunks that server needs)       │
  │                                    │
  │◀── GET /sync/pull ────────────────│
  │   (Client downloads chunks it      │
  │    doesn't have)                   │
  │                                    │

Phase 3: COMMIT
───────────────────
Client                              Server
  │                                    │
  │── POST /sync/commit ─────────────▶│
  │   { device_id, new_vector_clock,   │
  │     file_updates: [{path,          │
  │     manifest_id, action}...] }     │
  │                                    │
  │   Server: atomic metadata update   │
  │   Update VFS tree                  │
  │   Update device sync state         │
  │   Broadcast via WebSocket          │
  │                                    │
  │◀── { success, new_merkle_root } ──│
  │                                    │
```

#### Conflict Resolution
```go
type Conflict struct {
    ID           string
    FilePath     string
    LocalVersion  *Manifest
    RemoteVersion *Manifest
    LocalClock   VectorClock
    RemoteClock  VectorClock
    DetectedAt   time.Time
    ResolvedAt   *time.Time
    Resolution   string // "local", "remote", "rename", "merge", ""
}

type ConflictResolver struct{}

// Detect determines if two updates conflict.
func (cr *ConflictResolver) Detect(localClock, remoteClock VectorClock) bool {
    return localClock.Compare(remoteClock) == Concurrent
}

// AutoResolve attempts automatic resolution:
// - Different files → no conflict
// - Same file, non-overlapping chunk changes → merge manifests
// - Same file, overlapping changes → create conflict copy
func (cr *ConflictResolver) AutoResolve(conflict *Conflict) (*Resolution, error)

type Resolution struct {
    Strategy   string   // "auto_merge", "conflict_copy", "manual"
    WinnerID   string   // Manifest ID of winner (if auto)
    CopyPath   string   // Path of conflict copy (if conflict_copy)
}
```

### Module 5: WebDAV Server (`internal/webdav/`)

#### Method Dispatch
```go
type Handler struct {
    vfs    *vfs.VFS
    auth   *auth.Service
    crypto *crypto.Engine
    store  storage.Backend
}

// ServeHTTP handles all WebDAV requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "OPTIONS":    h.handleOptions(w, r)
    case "PROPFIND":   h.handlePropfind(w, r)
    case "PROPPATCH":  h.handleProppatch(w, r)
    case "MKCOL":      h.handleMkcol(w, r)
    case "GET", "HEAD": h.handleGet(w, r)
    case "PUT":        h.handlePut(w, r)
    case "DELETE":     h.handleDelete(w, r)
    case "COPY":       h.handleCopy(w, r)
    case "MOVE":       h.handleMove(w, r)
    case "LOCK":       h.handleLock(w, r)
    case "UNLOCK":     h.handleUnlock(w, r)
    }
}
```

#### Lock Manager (Class 2)
```go
type LockManager struct {
    mu    sync.RWMutex
    locks map[string]*Lock // path → lock
}

type Lock struct {
    Token    string
    Path     string
    Owner    string
    Type     string        // "write"
    Scope    string        // "exclusive" or "shared"
    Depth    string        // "0" or "infinity"
    Timeout  time.Duration
    Created  time.Time
}
```

### Module 6: Authentication & RBAC (`internal/auth/`)

#### JWT Structure
```go
type AccessClaims struct {
    UserID   string   `json:"uid"`
    Username string   `json:"usr"`
    Roles    []string `json:"roles"`
    DeviceID string   `json:"did"`
    jwt.RegisteredClaims
}

type RefreshClaims struct {
    SessionID string `json:"sid"`
    jwt.RegisteredClaims
}
```

#### TOTP (RFC 6238)
```go
// GenerateSecret creates a new TOTP secret.
func GenerateSecret() (secret string, qrURL string, err error)

// Validate checks a 6-digit TOTP code.
// Allows ±1 time step for clock drift tolerance.
func Validate(secret string, code string) bool
```

#### RBAC Enforcement
```go
// Authorize checks if user has permission for action on resource.
func (s *Service) Authorize(ctx context.Context, userID string, resource, action, scope string) error

// Middleware returns HTTP middleware that checks permissions.
func (s *Service) Middleware(resource, action string) func(http.Handler) http.Handler
```

### Module 7: Sharing Engine (`internal/share/`)

#### Public Share Link
```go
type ShareLink struct {
    ID            string
    FileID        string
    CreatedBy     string
    Token         string     // 32-char URL-safe random string
    PasswordHash  string     // Optional Argon2id hash
    ExpiresAt     *time.Time
    MaxDownloads  *int
    DownloadCount int
    AllowUpload   bool
    PreviewOnly   bool
    IsActive      bool
}

// Public share URL format: https://vault.example.com/s/{token}
// With password: POST /s/{token}/verify → set session cookie → access file
```

#### User Share (E2E)
```go
type UserShare struct {
    ID          string
    FileID      string
    CreatedBy   string
    SharedWith  string
    Permission  string   // read, write, manage
    EncryptedKey []byte  // File key re-encrypted with recipient's public key
}
```

### Module 8: Notification Hub (`internal/notify/`)

#### WebSocket Hub
```go
type Hub struct {
    mu         sync.RWMutex
    clients    map[string]map[*Client]bool // userID → connected clients
    broadcast  chan *Event
    register   chan *Client
    unregister chan *Client
}

type Event struct {
    Type      string      `json:"type"`      // file.created, file.updated, file.deleted, sync.complete, share.new
    UserID    string      `json:"user_id"`
    Path      string      `json:"path,omitempty"`
    Data      interface{} `json:"data,omitempty"`
    Timestamp time.Time   `json:"timestamp"`
}

type Client struct {
    UserID   string
    DeviceID string
    Conn     net.Conn     // Upgraded WebSocket connection
    Send     chan []byte
}
```

---

## 🔄 Critical Flows

### Flow 1: File Upload (Chunked)

```
Client                          Server
  │                                │
  │── POST /upload/init ─────────▶│ 1. Create upload session
  │   { path, size, encrypted }    │    Generate session_id
  │                                │    Validate quota
  │◀── { session_id, chunk_size } │
  │                                │
  │ [Split file into CDC chunks]   │
  │ [Encrypt each chunk if E2E]    │
  │                                │
  │── POST /upload/chunk ────────▶│ 2. For each chunk:
  │   { session_id, index,         │    Check dedup (hash exists?)
  │     hash, data }               │    If new: store in backend
  │                                │    If exists: increment ref_count
  │◀── { stored: true/deduped }   │
  │   ... repeat for all chunks    │
  │                                │
  │── POST /upload/complete ─────▶│ 3. Finalize:
  │   { session_id,                │    Create manifest
  │     encrypted_key }            │    Update file metadata in VFS
  │                                │    Update user used_bytes
  │◀── { file_id, version }       │    Broadcast change event
  │                                │    Log audit entry
```

### Flow 2: E2E File Sharing

```
User A (Owner)                    Server                    User B (Recipient)
  │                                  │                          │
  │ 1. Get User B's public key       │                          │
  │── GET /users/B/public-key ─────▶│                          │
  │◀── { public_key_b } ───────────│                          │
  │                                  │                          │
  │ 2. Re-encrypt file key           │                          │
  │ shared_key = X25519(             │                          │
  │   file_key, A_private, B_public) │                          │
  │                                  │                          │
  │── POST /share/user ────────────▶│ 3. Store share record    │
  │   { file_id, user_id_b,         │    with encrypted_key    │
  │     permission, encrypted_key }  │                          │
  │                                  │── WebSocket notify ─────▶│
  │                                  │   "User A shared X"      │
  │                                  │                          │
  │                                  │   4. User B accesses:    │
  │                                  │◀── GET /share/received ──│
  │                                  │──▶ { shares: [...] }     │
  │                                  │                          │
  │                                  │   5. Decrypt file key:   │
  │                                  │   file_key = X25519(     │
  │                                  │     encrypted_key,       │
  │                                  │     B_private, A_public) │
  │                                  │                          │
  │                                  │   6. Download & decrypt  │
  │                                  │◀── GET /download/{path} ─│
  │                                  │──▶ encrypted chunks      │
  │                                  │   Decrypt with file_key  │
```

### Flow 3: Sync Daemon Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Sync Daemon                            │
│                                                              │
│  ┌──────────┐    ┌─────────────┐    ┌──────────────────┐   │
│  │ FS Watch │───▶│ Change Queue│───▶│ Sync Scheduler   │   │
│  │ (inotify)│    │  (debounce  │    │ (batch changes,  │   │
│  └──────────┘    │   500ms)    │    │  throttle)       │   │
│                  └─────────────┘    └────────┬─────────┘   │
│                                              │              │
│  ┌──────────────────────────────────────────▼────────────┐ │
│  │                  Sync Loop                              │ │
│  │                                                         │ │
│  │  1. Build local Merkle tree                             │ │
│  │  2. POST /sync/negotiate (send root hash)               │ │
│  │  3. Receive diff from server                            │ │
│  │  4. Check for conflicts (vector clock comparison)       │ │
│  │  5. Upload new local chunks (POST /sync/push)           │ │
│  │  6. Download new remote chunks (GET /sync/pull)          │ │
│  │  7. Apply remote changes to local filesystem            │ │
│  │  8. POST /sync/commit (update vector clock)             │ │
│  │  9. Update local Merkle tree                            │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌──────────────┐                                           │
│  │  WebSocket   │ Real-time push from server                │
│  │  Listener    │ → Triggers immediate sync cycle           │
│  └──────────────┘                                           │
│                                                              │
│  ┌──────────────┐                                           │
│  │  Poll Timer  │ Fallback: every 30s (configurable)        │
│  │  (backup)    │ In case WebSocket disconnects              │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
```

### Flow 4: Conflict Detection & Resolution

```
Device A edits file.txt                Device B edits file.txt
at T=1, clock: {A:5, B:3}             at T=2, clock: {A:4, B:4}
         │                                       │
         ▼                                       ▼
    Push to server                          Push to server
         │                                       │
         ▼                                       ▼
┌─────────────────────────────────────────────────────────┐
│ Server receives both:                                    │
│                                                          │
│ Compare clocks:                                          │
│   A's clock: {A:5, B:3}                                 │
│   B's clock: {A:4, B:4}                                 │
│                                                          │
│   A:5 > A:4 but B:3 < B:4 → CONCURRENT → CONFLICT      │
│                                                          │
│ Auto-resolve attempt:                                    │
│   Compare chunk manifests:                               │
│   A changed chunks [3, 5]                                │
│   B changed chunks [7, 8]                                │
│   No overlap → AUTO MERGE possible                       │
│                                                          │
│   Create merged manifest: [1,2,A3,4,A5,6,B7,B8,9...]   │
│   Merged clock: {A:5, B:4}                              │
│                                                          │
│ If overlap → Create conflict copy:                       │
│   file.txt (keep first-committed version)                │
│   file.conflict-B-20260330T142030Z.txt (second version) │
│   Notify both devices of conflict                        │
└─────────────────────────────────────────────────────────┘
```

---

## 🖥️ Server Startup Sequence

```
main()
  │
  ├── 1. Parse CLI flags (--config, --init, serve)
  ├── 2. Load config (YAML + env overrides)
  ├── 3. Validate config
  ├── 4. Initialize CobaltDB
  │      ├── Open/create database file
  │      ├── Run migrations (schema.go)
  │      └── Seed default roles (admin, user, guest)
  ├── 5. Initialize storage backend (local or S3)
  │      └── Verify connectivity (S3: HeadBucket)
  ├── 6. Initialize crypto engine
  ├── 7. Initialize VFS layer
  ├── 8. Initialize sync engine
  ├── 9. Initialize auth service
  ├── 10. Initialize share service
  ├── 11. Initialize notification hub
  ├── 12. Build HTTP router
  │       ├── /api/v1/* → REST API handlers
  │       ├── /dav/*    → WebDAV handler
  │       ├── /s/*      → Public share handler
  │       ├── /ws       → WebSocket upgrade
  │       └── /*        → Embedded Web UI (SPA fallback)
  ├── 13. Apply middleware stack:
  │       ├── Recovery (panic handler)
  │       ├── Request ID
  │       ├── Logging
  │       ├── CORS
  │       ├── Security headers
  │       ├── Rate limiting
  │       ├── Gzip compression
  │       └── Auth (JWT extraction)
  ├── 14. Configure TLS (ACME or manual)
  ├── 15. Start HTTP/2 server
  ├── 16. Start background workers:
  │       ├── Chunk garbage collector (hourly)
  │       ├── Session cleanup (every 15m)
  │       ├── Trash auto-purge (daily)
  │       ├── Thumbnail generator (on-demand queue)
  │       └── Audit log rotation (weekly)
  ├── 17. Register signal handlers (SIGTERM, SIGINT)
  └── 18. Block until shutdown signal
          ├── Graceful HTTP shutdown (30s timeout)
          ├── Close WebSocket connections
          ├── Flush pending writes
          └── Close CobaltDB
```

---

## 🧰 Build & Release

### Makefile Targets
```makefile
VERSION    := $(shell git describe --tags --always)
LDFLAGS    := -s -w -X main.version=$(VERSION)

.PHONY: build build-all clean test lint

build:                          # Build server binary
	go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift ./cmd/vaultdrift

build-cli:                      # Build CLI client
	go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-cli ./cmd/vaultdrift-cli

build-desktop:                  # Build desktop tray app
	go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-desktop ./cmd/vaultdrift-desktop

build-all:                      # Cross-compile all platforms
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-linux-amd64 ./cmd/vaultdrift
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-linux-arm64 ./cmd/vaultdrift
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-darwin-amd64 ./cmd/vaultdrift
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-darwin-arm64 ./cmd/vaultdrift
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/vaultdrift-windows-amd64.exe ./cmd/vaultdrift

test:
	go test -race -coverprofile=coverage.out ./...

test-integration:
	go test -tags=integration -race ./...

bench:
	go test -bench=. -benchmem ./internal/chunk/ ./internal/crypto/ ./internal/sync/

lint:
	go vet ./...
	staticcheck ./...

clean:
	rm -rf bin/ coverage.out

docker:
	docker build -t vaultdrift/vaultdrift:$(VERSION) .
```

### Dockerfile
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -ldflags "-s -w" -o vaultdrift ./cmd/vaultdrift

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/vaultdrift /usr/local/bin/vaultdrift
EXPOSE 8443
VOLUME /var/lib/vaultdrift
ENTRYPOINT ["vaultdrift"]
CMD ["serve"]
```

---

## 📐 Error Handling Strategy

### Error Types
```go
type AppError struct {
    Code    int    `json:"code"`              // HTTP status code
    Type    string `json:"type"`              // Machine-readable error type
    Message string `json:"message"`           // Human-readable message
    Details any    `json:"details,omitempty"` // Additional context
}

// Standard error types:
// auth.invalid_credentials, auth.token_expired, auth.insufficient_permission
// fs.not_found, fs.already_exists, fs.quota_exceeded, fs.name_invalid
// sync.conflict, sync.device_not_found, sync.stale_state
// share.not_found, share.expired, share.password_required, share.download_limit
// storage.backend_error, storage.chunk_not_found
// crypto.decryption_failed, crypto.invalid_key
// system.internal_error, system.rate_limited
```

### API Response Format
```json
// Success
{
    "data": { ... },
    "meta": {
        "request_id": "req_abc123",
        "timestamp": "2026-03-30T14:20:00Z"
    }
}

// Error
{
    "error": {
        "code": 409,
        "type": "sync.conflict",
        "message": "File modified on another device",
        "details": {
            "conflict_id": "conf_xyz",
            "local_version": 3,
            "remote_version": 4
        }
    },
    "meta": {
        "request_id": "req_abc123",
        "timestamp": "2026-03-30T14:20:00Z"
    }
}
```

---

*Implementation follows zero-dependency philosophy: every component from S3 signing to TOTP generation to Rabin fingerprinting built from scratch in pure Go.*
