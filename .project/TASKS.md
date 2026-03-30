# VaultDrift — TASKS.md

> **UI Stack Update**: Web UI will now be built with **React 19 + Tailwind CSS 4.1 + Shadcn UI + Lucide React** instead of Alpine.js. Responsive, dark/light theme. Build output embedded in binary via `go:embed`.

---

## Phase 0 — Project Scaffolding & Foundation

### Task 0.1: Repository & Go Module Setup
```
Priority: P0 | Effort: S | Dependencies: None
```
- [ ] `go mod init github.com/vaultdrift/vaultdrift`
- [ ] Create directory structure: `cmd/`, `internal/`, `client/`, `desktop/`, `web/`, `cobaltdb/`
- [ ] `cmd/vaultdrift/main.go` — Placeholder server entry
- [ ] `cmd/vaultdrift-cli/main.go` — Placeholder CLI entry
- [ ] `cmd/vaultdrift-desktop/main.go` — Placeholder desktop entry
- [ ] `Makefile` — build, test, lint, clean, docker targets
- [ ] `.gitignore` — Go + Node + IDE patterns
- [ ] `Dockerfile` — Multi-stage build (Go builder + Alpine runtime)
- [ ] `vaultdrift.yaml.example` — Example config with all options documented
- [ ] `README.md` — Project description, quick start, badges
- [ ] `LICENSE` — Apache 2.0

### Task 0.2: Configuration System (`internal/config/`)
```
Priority: P0 | Effort: M | Dependencies: None
```
- [ ] `config.go` — Config struct with all sections (server, storage, database, auth, sync, encryption, sharing, users, smtp, logging)
- [ ] `loader.go` — YAML file loader with `VAULTDRIFT_` environment variable overrides
- [ ] `validate.go` — Config validation (required fields, valid ranges, path existence)
- [ ] Defaults: port 8443, TLS enabled, local storage, 10GB quota, 15m access token TTL
- [ ] Config hot-reload support (SIGHUP signal)
- [ ] Unit tests: load, validate, env override, defaults

### Task 0.3: Utility Package (`internal/util/`)
```
Priority: P0 | Effort: S | Dependencies: None
```
- [ ] `hash.go` — SHA-256 helpers (HashBytes, HashReader, HashString)
- [ ] `encoding.go` — Base64 URL-safe encode/decode, hex helpers
- [ ] `sanitize.go` — Path normalization, filename sanitization, traversal prevention
- [ ] `size.go` — Human-readable size formatting (1.5 GB, 340 KB)
- [ ] `time.go` — RFC 3339 formatting, relative time ("2 hours ago")
- [ ] `pool.go` — `sync.Pool` based byte buffer pool (4KB, 64KB, 1MB, 4MB tiers)
- [ ] `id.go` — UUID v7 generator (timestamp-sortable, pure Go)
- [ ] Unit tests: 100% coverage for all helpers

---

## Phase 1 — CobaltDB Integration & Data Layer

### Task 1.1: CobaltDB Integration (`internal/db/`)
```
Priority: P0 | Effort: L | Dependencies: 0.1
```
- [ ] Integrate CobaltDB as git subtree or Go module under `cobaltdb/`
- [ ] `db.go` — Database manager: Open, Close, migration runner
- [ ] `schema.go` — All table CREATE statements (from IMPLEMENTATION.md)
- [ ] Migration system: version tracking, up/down migration support
- [ ] Default seed data: admin/user/guest roles, default permissions
- [ ] Connection pool / concurrent access management
- [ ] Unit tests: Open, migrate, seed, CRUD basics

### Task 1.2: User Data Access (`internal/db/users.go`)
```
Priority: P0 | Effort: M | Dependencies: 1.1
```
- [ ] `CreateUser(user *User) error`
- [ ] `GetUserByID(id string) (*User, error)`
- [ ] `GetUserByUsername(username string) (*User, error)`
- [ ] `GetUserByEmail(email string) (*User, error)`
- [ ] `UpdateUser(id string, updates map[string]any) error`
- [ ] `DeleteUser(id string) error`
- [ ] `ListUsers(offset, limit int, filter UserFilter) ([]*User, int, error)`
- [ ] `UpdateUsedBytes(id string, delta int64) error`
- [ ] Unit tests: CRUD, unique constraints, filtering

### Task 1.3: File Metadata Data Access (`internal/db/files.go`)
```
Priority: P0 | Effort: M | Dependencies: 1.1
```
- [ ] `CreateFile(file *File) error`
- [ ] `GetFileByID(id string) (*File, error)`
- [ ] `GetFileByPath(userID, parentID, name string) (*File, error)`
- [ ] `ListDirectory(userID, parentID string, opts ListOpts) ([]*File, error)`
- [ ] `UpdateFile(id string, updates map[string]any) error`
- [ ] `MoveFile(id, newParentID, newName string) error`
- [ ] `SoftDelete(id string) error` — Set is_trashed
- [ ] `ListTrash(userID string) ([]*File, error)`
- [ ] `RestoreFromTrash(id string) error`
- [ ] `PermanentDelete(id string) error`
- [ ] `SearchFiles(userID, query string, limit int) ([]*File, error)` — Filename search
- [ ] `RecentFiles(userID string, limit int) ([]*File, error)` — By updated_at
- [ ] Unit tests: Full CRUD, tree operations, trash lifecycle, search

### Task 1.4: Chunk Index Data Access (`internal/db/chunks.go`)
```
Priority: P0 | Effort: S | Dependencies: 1.1
```
- [ ] `CreateChunk(chunk *Chunk) error`
- [ ] `GetChunk(hash string) (*Chunk, error)`
- [ ] `ChunkExists(hash string) (bool, error)`
- [ ] `IncrementRefCount(hash string) error`
- [ ] `DecrementRefCount(hash string) error`
- [ ] `ListOrphanedChunks() ([]*Chunk, error)` — ref_count = 0
- [ ] `DeleteChunk(hash string) error`
- [ ] Unit tests: Ref counting, orphan detection

### Task 1.5: Manifest Data Access (`internal/db/manifests.go`)
```
Priority: P0 | Effort: S | Dependencies: 1.1
```
- [ ] `CreateManifest(manifest *Manifest) error`
- [ ] `GetManifest(id string) (*Manifest, error)`
- [ ] `GetLatestManifest(fileID string) (*Manifest, error)`
- [ ] `ListVersions(fileID string) ([]*Manifest, error)`
- [ ] `DeleteManifest(id string) error`

### Task 1.6: Remaining DB Modules
```
Priority: P0 | Effort: M | Dependencies: 1.1
```
- [ ] `sessions.go` — Session CRUD, cleanup expired, list by user
- [ ] `shares.go` — Share CRUD, list by file, list by user, token lookup
- [ ] `syncstate.go` — Device CRUD, sync state upsert, vector clock update
- [ ] `audit.go` — Append audit entry, list with filters (user, action, date range)
- [ ] `settings.go` — Key-value settings CRUD
- [ ] Unit tests per module

---

## Phase 2 — Storage & Chunking Engine

### Task 2.1: Storage Backend Interface (`internal/storage/storage.go`)
```
Priority: P0 | Effort: S | Dependencies: 0.3
```
- [ ] Define `Backend` interface: Put, Get, Delete, Exists, List, Stats
- [ ] `StorageStats` struct
- [ ] `NewBackend(cfg config.Storage) (Backend, error)` — Factory function

### Task 2.2: Local Filesystem Backend (`internal/storage/local.go`)
```
Priority: P0 | Effort: M | Dependencies: 2.1
```
- [ ] Key → path mapping: `{dataDir}/chunks/{hash[:2]}/{hash[2:]}.chunk`
- [ ] `Put` — Atomic write (temp file → rename), create prefix dir
- [ ] `Get` — Read file, return bytes
- [ ] `Delete` — Remove file, cleanup empty parent dir
- [ ] `Exists` — `os.Stat` check
- [ ] `List` — Walk prefix directory
- [ ] `Stats` — Walk all, sum sizes
- [ ] Concurrent safety: Per-key file locking not needed (atomic rename)
- [ ] Unit tests: Full CRUD, concurrent writes, edge cases (missing dir, permission errors)

### Task 2.3: S3 Client from Scratch (`internal/storage/s3client/`)
```
Priority: P0 | Effort: XL | Dependencies: 0.3
```
- [ ] `client.go` — HTTP client wrapper, endpoint config, retries
- [ ] `sign.go` — AWS Signature V4 implementation:
  - Canonical request building
  - String-to-sign construction
  - Signing key derivation (HMAC-SHA256 chain)
  - Authorization header generation
  - Chunked transfer encoding signing (for streaming uploads)
- [ ] `operations.go`:
  - `PutObject(bucket, key string, data io.Reader, size int64) error`
  - `GetObject(bucket, key string) (io.ReadCloser, int64, error)`
  - `DeleteObject(bucket, key string) error`
  - `HeadObject(bucket, key string) (*ObjectInfo, error)`
  - `ListObjectsV2(bucket, prefix string) ([]ObjectInfo, error)`
  - `HeadBucket(bucket string) error` — Connectivity check
- [ ] `multipart.go` — Multipart upload for large objects:
  - `CreateMultipartUpload`
  - `UploadPart`
  - `CompleteMultipartUpload`
  - `AbortMultipartUpload`
- [ ] Retry: Exponential backoff (3 attempts, 1s/2s/4s)
- [ ] Connection pooling via `http.Transport` (MaxIdleConns, IdleConnTimeout)
- [ ] Unit tests: Signing test vectors (AWS official test suite), mock HTTP server tests
- [ ] Integration tests: MinIO container test (build tag `integration`)

### Task 2.4: S3 Storage Backend (`internal/storage/s3.go`)
```
Priority: P0 | Effort: M | Dependencies: 2.1, 2.3
```
- [ ] `s3Backend` struct implementing `Backend` interface
- [ ] Key format: `chunks/{hash[:2]}/{hash[2:]}.chunk`
- [ ] `Put` — PutObject (auto multipart for >5MB)
- [ ] `Get` — GetObject, stream to bytes
- [ ] `Delete` — DeleteObject
- [ ] `Exists` — HeadObject
- [ ] `List` — ListObjectsV2 with prefix
- [ ] `Stats` — ListObjectsV2, sum sizes
- [ ] Unit tests: Mock S3 server

### Task 2.5: Content-Defined Chunking Engine (`internal/chunk/`)
```
Priority: P0 | Effort: L | Dependencies: 0.3
```
- [ ] `rabin.go` — Rabin fingerprint rolling hash:
  - 48-byte sliding window
  - Polynomial modular arithmetic
  - Pre-computed lookup tables for O(1) slide
- [ ] `chunker.go` — CDC implementation:
  - `NewChunker(min, avg, max int)` — Configurable sizes
  - `Chunk(r io.Reader) ([]ChunkInfo, error)` — Split reader into chunks
  - Boundary detection: `hash & mask == mask`
  - Min/max enforcement
  - SHA-256 hash per chunk
  - Memory efficient: Stream processing, reuse buffers via sync.Pool
- [ ] `manifest.go` — Manifest struct, JSON serialization
- [ ] `dedup.go` — Check chunk existence before storage
- [ ] `reassemble.go` — `Reassemble(manifest, store, writer)` — Stream chunks in order
- [ ] Benchmarks: Throughput (target: >500MB/s chunking on single core)
- [ ] Unit tests: Known test vectors, edge cases (tiny files, exact boundary files, max chunk files)

---

## Phase 3 — Encryption Engine

### Task 3.1: Core Cryptographic Primitives (`internal/crypto/`)
```
Priority: P0 | Effort: L | Dependencies: 0.3
```
- [ ] `random.go` — `crypto/rand` wrappers: RandomBytes(n), RandomString(n, alphabet)
- [ ] `aes.go` — AES-256-GCM:
  - `Encrypt(plaintext, key []byte) ([]byte, error)` — Returns nonce+ciphertext+tag
  - `Decrypt(ciphertext, key []byte) ([]byte, error)`
  - Random 12-byte nonce per encryption
  - Streaming encrypt/decrypt for large data (chunk-by-chunk)
- [ ] `argon2.go` — Argon2id key derivation:
  - `DeriveKey(passphrase string, salt []byte) [32]byte`
  - Pure Go implementation (golang.org/x/crypto style but stdlib-only)
  - Parameters: time=3, memory=64MB, threads=4, keyLen=32
  - `GenerateSalt() []byte` — 16-byte random salt
- [ ] `x25519.go` — X25519 Diffie-Hellman:
  - `GenerateKeyPair() (pub, priv [32]byte, error)`
  - `SharedSecret(myPrivate, theirPublic [32]byte) [32]byte`
  - Pure Go Curve25519 scalar multiplication
- [ ] Unit tests: Known-answer test vectors (NIST, RFC), round-trip tests

### Task 3.2: Key Management (`internal/crypto/keys.go`)
```
Priority: P0 | Effort: M | Dependencies: 3.1
```
- [ ] `GenerateFileKey() ([32]byte, error)` — Random 256-bit key
- [ ] `WrapKey(fileKey, masterKey [32]byte) ([]byte, error)` — AES-256-GCM wrap
- [ ] `UnwrapKey(wrapped []byte, masterKey [32]byte) ([32]byte, error)`
- [ ] `ShareFileKey(fileKey, senderPrivate, recipientPublic [32]byte) ([]byte, error)` — X25519 shared secret → AES wrap
- [ ] `ReceiveFileKey(encrypted []byte, recipientPrivate, senderPublic [32]byte) ([32]byte, error)`
- [ ] Unit tests: Wrap/unwrap round-trip, cross-user sharing round-trip

### Task 3.3: Recovery Key System (`internal/crypto/recovery.go`)
```
Priority: P1 | Effort: M | Dependencies: 3.1
```
- [ ] BIP39 word list embed (2048 English words)
- [ ] `GenerateRecoveryKey(masterKey [32]byte) (mnemonic string, error)` — Encode master key as 24 words
- [ ] `RecoverMasterKey(mnemonic string) ([32]byte, error)` — Decode 24 words back to master key
- [ ] Checksum verification (last word includes checksum bits)
- [ ] Unit tests: Round-trip, invalid mnemonic detection

### Task 3.4: Encrypted Chunking Integration
```
Priority: P0 | Effort: M | Dependencies: 2.5, 3.1, 3.2
```
- [ ] `ChunkEncrypted(r io.Reader, fileKey []byte) ([]ChunkInfo, [][]byte, error)` — Chunk then encrypt each chunk
- [ ] `ReassembleDecrypt(manifest, store, fileKey, writer)` — Fetch → decrypt → write in order
- [ ] Chunk hash computed on plaintext (before encryption) for dedup across encrypted files
- [ ] Unit tests: Upload encrypted → download decrypted round-trip

---

## Phase 4 — Authentication & Authorization

### Task 4.1: Password Hashing (`internal/auth/password.go`)
```
Priority: P0 | Effort: S | Dependencies: 3.1
```
- [ ] `HashPassword(password string) (string, error)` — Argon2id with random salt, encode as PHC string
- [ ] `VerifyPassword(password, hash string) (bool, error)` — Parse PHC string, verify
- [ ] Timing-safe comparison
- [ ] Unit tests: Hash/verify round-trip, wrong password, format validation

### Task 4.2: JWT Token System (`internal/auth/jwt.go`)
```
Priority: P0 | Effort: M | Dependencies: 0.2
```
- [ ] Pure Go JWT implementation (no third-party library):
  - Header: `{"alg":"HS256","typ":"JWT"}`
  - Base64URL encode/decode
  - HMAC-SHA256 signing/verification
- [ ] `GenerateAccessToken(claims AccessClaims) (string, error)` — 15m TTL
- [ ] `GenerateRefreshToken(claims RefreshClaims) (string, error)` — 7d TTL
- [ ] `ValidateAccessToken(token string) (*AccessClaims, error)`
- [ ] `ValidateRefreshToken(token string) (*RefreshClaims, error)`
- [ ] Token rotation: Refresh token is single-use, new pair on each refresh
- [ ] Unit tests: Generate/validate, expired token, tampered token, rotation

### Task 4.3: TOTP Implementation (`internal/auth/totp.go`)
```
Priority: P1 | Effort: M | Dependencies: 3.1
```
- [ ] RFC 6238 TOTP from scratch:
  - HMAC-SHA1 based OTP
  - 30-second time step
  - 6-digit codes
  - ±1 step tolerance for clock drift
- [ ] `GenerateSecret() (secret, otpauthURL string, error)` — 20-byte random secret, otpauth:// URI
- [ ] `ValidateCode(secret, code string) bool`
- [ ] QR code data generation (otpauth://totp/VaultDrift:{username}?secret={secret}&issuer=VaultDrift)
- [ ] Backup codes: Generate 10 one-time-use recovery codes
- [ ] Unit tests: Known test vectors (RFC 6238 Appendix B), time drift

### Task 4.4: Auth Service (`internal/auth/auth.go`)
```
Priority: P0 | Effort: M | Dependencies: 4.1, 4.2, 1.2, 1.6
```
- [ ] `Login(username, password string) (*TokenPair, error)`:
  - Verify credentials
  - Check account status (active/disabled/locked)
  - Brute-force check (progressive delay)
  - Create session record
  - Generate JWT pair
- [ ] `Refresh(refreshToken string) (*TokenPair, error)`:
  - Validate refresh token
  - Check session still valid
  - Rotate: invalidate old refresh, generate new pair
- [ ] `Logout(sessionID string) error` — Invalidate session
- [ ] `LoginWithTOTP(username, password, code string) (*TokenPair, error)`
- [ ] Brute-force protection: Track failed attempts per username, progressive delay (1s, 2s, 4s, 8s...), lockout after 5 attempts for 15m
- [ ] Unit tests: Full login flow, brute-force, token rotation

### Task 4.5: RBAC Engine (`internal/auth/rbac.go`)
```
Priority: P0 | Effort: M | Dependencies: 1.6
```
- [ ] `Authorize(ctx, userID, resource, action, scope string) error`
- [ ] Permission check logic:
  1. Get user's roles
  2. Get all permissions for those roles
  3. Check if any permission matches (resource, action, scope)
  4. Scope check: "own" = only user's resources, "group" = shared resources, "all" = everything
- [ ] Default roles seed:
  - **admin**: all resources, all actions, scope=all
  - **user**: file/folder/share (own), read (group), no user/system manage
  - **guest**: file/folder read (group only)
- [ ] Permission cache: In-memory cache per user, invalidate on role change
- [ ] Unit tests: Permission matrix, scope enforcement, cache invalidation

### Task 4.6: API Token System (`internal/auth/token.go`)
```
Priority: P1 | Effort: S | Dependencies: 1.6, 4.5
```
- [ ] `GenerateAPIToken(userID, name string, permissions []string) (token string, error)` — SHA-256 hash stored, raw returned once
- [ ] `ValidateAPIToken(token string) (*User, []string, error)` — Lookup hash, return user + permissions
- [ ] Scoped permissions: Token can have subset of user's permissions
- [ ] Expiry: Optional expiration date
- [ ] Unit tests: Generate, validate, expired, revoked

### Task 4.7: Auth Middleware (`internal/server/middleware.go` — auth part)
```
Priority: P0 | Effort: M | Dependencies: 4.2, 4.5, 4.6
```
- [ ] JWT extraction from `Authorization: Bearer <token>` header
- [ ] API token extraction from `Authorization: Token <token>` header
- [ ] Cookie-based session for Web UI (HttpOnly, Secure, SameSite=Strict)
- [ ] Request context injection: UserID, Roles, DeviceID
- [ ] RBAC middleware factory: `RequirePermission(resource, action string) Middleware`
- [ ] Public route bypass (share links, login, static assets)
- [ ] Unit tests: Each auth method, missing/invalid/expired tokens

---

## Phase 5 — Virtual Filesystem & Core API

### Task 5.1: VFS Layer (`internal/vfs/`)
```
Priority: P0 | Effort: L | Dependencies: 1.3, 1.4, 1.5
```
- [ ] `vfs.go` — VFS struct, core operations interface:
  - `MkDir(userID, parentPath, name string) (*File, error)`
  - `CreateFile(userID, parentPath, name string, manifest *Manifest, encrypted bool, encKey []byte) (*File, error)`
  - `GetByPath(userID, path string) (*File, error)` — Path resolution (split, walk tree)
  - `Move(userID, fileID, newParentPath, newName string) error`
  - `Copy(userID, fileID, destParentPath, destName string) (*File, error)`
  - `Delete(userID, fileID string) error` — Soft delete to trash
  - `ListDir(userID, path string, opts ListOpts) ([]*File, error)`
  - `Search(userID, query string) ([]*File, error)`
  - `Recent(userID string, limit int) ([]*File, error)`
- [ ] `path.go` — Path utilities:
  - Normalize: `//foo/../bar` → `/bar`
  - Split: `/a/b/c` → `["a", "b", "c"]`
  - Validate: No null bytes, no `.`, no `..`, max 255 chars per component
  - Join: Components → full path
- [ ] `tree.go` — Path-to-ID resolution via DB lookups (root → parent chain)
- [ ] `trash.go` — Trash operations: list, restore (check name conflict), purge, auto-purge (30 days)
- [ ] `search.go` — Filename LIKE search, recent files query
- [ ] Unit tests: Path resolution, tree traversal, move/copy edge cases, trash lifecycle

### Task 5.2: HTTP Server Foundation (`internal/server/`)
```
Priority: P0 | Effort: M | Dependencies: 0.2
```
- [ ] `server.go` — Server struct:
  - `New(cfg *config.Config) (*Server, error)` — Initialize all services
  - `Start() error` — Listen, serve HTTP/2 + TLS
  - `Shutdown(ctx context.Context) error` — Graceful shutdown
- [ ] `router.go` — Route registration:
  - Custom router (stdlib `http.ServeMux` based, Go 1.22+ pattern matching)
  - Route groups: `/api/v1/`, `/dav/`, `/s/`, `/ws`, `/*`
  - Method-aware routing: `GET /api/v1/fs/list`
- [ ] `middleware.go` — Middleware stack:
  - Recovery (panic → 500)
  - Request ID (UUID v7)
  - Access logging (method, path, status, duration)
  - CORS (configurable origins)
  - Security headers (HSTS, CSP, X-Frame-Options, X-Content-Type-Options)
  - Rate limiting (token bucket, per-IP + per-user)
  - Gzip compression (Accept-Encoding aware)
- [ ] `context.go` — Request context helpers: GetUserID, GetRoles, GetRequestID

### Task 5.3: File & Folder API Handlers (`internal/api/files.go`)
```
Priority: P0 | Effort: L | Dependencies: 5.1, 5.2, 4.7
```
- [ ] `GET /api/v1/fs/list?path=&sort=&order=&page=&limit=` — Directory listing with pagination
- [ ] `GET /api/v1/fs/info/{path}` — File/folder metadata (size, mime, version, modified, permissions)
- [ ] `POST /api/v1/fs/mkdir` — Create directory, validate path, check parent exists
- [ ] `PUT /api/v1/fs/rename` — Rename/move, check destination conflicts
- [ ] `DELETE /api/v1/fs/delete` — Soft delete to trash
- [ ] `POST /api/v1/fs/copy` — Deep copy (recursive for folders)
- [ ] `GET /api/v1/fs/search?q=&limit=` — Filename search
- [ ] `GET /api/v1/fs/recent?limit=` — Recent files
- [ ] `GET /api/v1/fs/trash` — List trash
- [ ] `POST /api/v1/fs/trash/restore` — Restore from trash
- [ ] `DELETE /api/v1/fs/trash/purge` — Permanent delete (purge all or specific)
- [ ] Consistent JSON response format: `{data, meta}` / `{error, meta}`
- [ ] RBAC checks on every endpoint
- [ ] Audit logging for mutating operations
- [ ] Unit tests: Each endpoint, auth checks, error cases

### Task 5.4: Upload System (`internal/api/upload.go`)
```
Priority: P0 | Effort: L | Dependencies: 2.5, 3.4, 5.1, 2.2/2.4
```
- [ ] `POST /api/v1/upload/init`:
  - Validate: path, filename, declared size
  - Quota check: user.used_bytes + size <= user.quota_bytes
  - Create upload session (in-memory, TTL 24h)
  - Return: session_id, recommended chunk_size
- [ ] `POST /api/v1/upload/chunk`:
  - Receive: session_id, chunk_index, chunk_hash, chunk_data (multipart or raw body)
  - Verify hash matches
  - Dedup check: chunk exists in DB? → skip storage, increment ref_count
  - If new: Encrypt chunk (if E2E enabled) → store in backend
  - Track progress in session
- [ ] `POST /api/v1/upload/complete`:
  - Verify all chunks received
  - Create manifest (ordered chunk list)
  - Create/update file metadata in VFS
  - Update user used_bytes
  - Broadcast change event via WebSocket
  - Audit log entry
  - Cleanup upload session
  - Return: file_id, version
- [ ] Simple upload shortcut: `POST /api/v1/upload/simple` — Single request for small files (<10MB)
- [ ] Upload resume: If session exists and partial chunks, continue from last
- [ ] Concurrent chunk uploads: Session tracks which chunks received (bitmap)
- [ ] Unit tests: Full upload flow, dedup, quota enforcement, resume, concurrent chunks

### Task 5.5: Download System (`internal/api/download.go`)
```
Priority: P0 | Effort: M | Dependencies: 2.5, 3.4, 5.1
```
- [ ] `GET /api/v1/download/{path}`:
  - Resolve path → file → manifest
  - Stream reassembly: Fetch chunks in order → decrypt if E2E → write to response
  - `Content-Disposition: attachment; filename="..."`
  - `Content-Length` header (known from manifest)
  - Range request support (HTTP 206 Partial Content) for resume/seek
  - ETag header (file checksum)
  - If-None-Match support (304 Not Modified)
- [ ] `GET /api/v1/download/zip?paths[]=`:
  - Accept multiple file/folder paths
  - On-the-fly ZIP streaming (no temp file)
  - Recursive folder inclusion
  - Progress tracking via headers or WebSocket
- [ ] `GET /api/v1/thumbnail/{path}`:
  - Check thumbnail cache
  - If miss: Generate thumbnail (Task 8.1)
  - Return WebP thumbnail (150x150 default, configurable)
- [ ] Unit tests: Stream download, range requests, ZIP generation, ETag caching

### Task 5.6: Auth API Handlers (`internal/api/auth.go`)
```
Priority: P0 | Effort: M | Dependencies: 4.4, 4.3
```
- [ ] `POST /api/v1/auth/login` — `{username, password}` → `{access_token, refresh_token, user}`
- [ ] `POST /api/v1/auth/login/totp` — `{username, password, code}` → tokens (if TOTP enabled)
- [ ] `POST /api/v1/auth/refresh` — `{refresh_token}` → new token pair
- [ ] `POST /api/v1/auth/logout` — Invalidate session
- [ ] `POST /api/v1/auth/totp/setup` — Generate TOTP secret + QR URL (auth required)
- [ ] `POST /api/v1/auth/totp/verify` — Verify code to enable TOTP
- [ ] `POST /api/v1/auth/totp/disable` — Disable TOTP (requires current code)
- [ ] Set HttpOnly cookie for Web UI on login (alongside JSON response)
- [ ] Unit tests: Full auth flows

### Task 5.7: User Management API (`internal/api/users.go`)
```
Priority: P0 | Effort: M | Dependencies: 1.2, 4.5, 4.7
```
- [ ] `GET /api/v1/users` — List users (admin only), pagination + search
- [ ] `POST /api/v1/users` — Create user (admin only), password validation
- [ ] `PUT /api/v1/users/{id}` — Update user (admin only), quota/role/status changes
- [ ] `DELETE /api/v1/users/{id}` — Delete user (admin only), cascade delete files/shares/sessions
- [ ] `GET /api/v1/users/me` — Current user profile
- [ ] `PUT /api/v1/users/me` — Update own profile (display name, email, password)
- [ ] `PUT /api/v1/users/me/password` — Change password (requires current password)
- [ ] `GET /api/v1/users/me/devices` — List own devices
- [ ] `DELETE /api/v1/users/me/devices/{id}` — Revoke device
- [ ] Unit tests: CRUD, permission checks, self-service vs admin

### Task 5.8: Admin API Handlers (`internal/api/admin.go`)
```
Priority: P1 | Effort: M | Dependencies: 5.7, 1.6
```
- [ ] `GET /api/v1/admin/stats` — System statistics:
  - Total users, active users, storage used/available
  - Dedup savings, chunk count, file count
  - Active sessions, active sync connections
- [ ] `PUT /api/v1/admin/settings` — Update system settings (stored in settings table)
- [ ] `GET /api/v1/admin/audit` — Audit log with filters: user, action, resource, date range, pagination
- [ ] `POST /api/v1/admin/roles` — Create custom role
- [ ] `PUT /api/v1/admin/roles/{id}` — Update role permissions
- [ ] `DELETE /api/v1/admin/roles/{id}` — Delete custom role (not system roles)
- [ ] Unit tests: Stats accuracy, settings persistence, audit filtering

---

## Phase 6 — Sharing System

### Task 6.1: Public Share Links (`internal/share/`, `internal/api/share.go`)
```
Priority: P0 | Effort: L | Dependencies: 5.1, 3.2, 4.5
```
- [ ] `POST /api/v1/share/link`:
  - Generate 32-char URL-safe random token
  - Optional: password (Argon2id hash), expiry date, max downloads, preview-only, allow upload
  - Store share record in DB
  - Return: share URL `{base_url}/s/{token}`
- [ ] `GET /api/v1/share/links` — List user's active share links with stats
- [ ] `PUT /api/v1/share/link/{id}` — Update share settings
- [ ] `DELETE /api/v1/share/link/{id}` — Revoke (set is_active=false)
- [ ] Public share handler: `GET /s/{token}`:
  - Validate token, check active, check expiry
  - If password: show password form → verify → set session cookie
  - If preview-only: render preview (images inline, PDFs embedded)
  - Download: Check download_count < max_downloads, increment counter
  - Folder share: Browse directory, download individual files or ZIP
- [ ] QR code generation (`internal/share/qrcode.go`): Pure Go QR encoder
- [ ] Unit tests: Create, access, password, expiry, download limits, folder browse

### Task 6.2: User-to-User Sharing (`internal/share/`)
```
Priority: P0 | Effort: M | Dependencies: 6.1, 3.2
```
- [ ] `POST /api/v1/share/user`:
  - Share file/folder with specific user
  - Permission: read, write, manage
  - E2E: Re-encrypt file key with recipient's public key
  - Notify recipient via WebSocket
- [ ] `GET /api/v1/share/received` — Files shared with current user
- [ ] `DELETE /api/v1/share/user/{id}` — Remove share
- [ ] Shared folder: Recipient sees in "Shared with me" virtual folder
- [ ] Write permission: Recipient can modify → chunks stored under owner's quota
- [ ] Unit tests: Share/unshare, permission enforcement, E2E key exchange

---

## Phase 7 — Sync Protocol

### Task 7.1: Vector Clock (`internal/sync/vector.go`)
```
Priority: P0 | Effort: S | Dependencies: None
```
- [ ] `VectorClock` type (map[string]uint64)
- [ ] `Increment(deviceID)`, `Merge(other)`, `Compare(other) Ordering`
- [ ] JSON marshal/unmarshal
- [ ] Unit tests: Compare all cases (Before, After, Concurrent, Equal), merge

### Task 7.2: Merkle Tree (`internal/sync/merkle.go`)
```
Priority: P0 | Effort: M | Dependencies: 1.3
```
- [ ] `Build(userID)` — Construct from file metadata in DB
- [ ] Node hash: Folders = SHA-256(sorted child hashes), Files = file checksum
- [ ] `Diff(local, remote)` — Compare trees, return added/modified/deleted paths
- [ ] Subtree optimization: If folder hashes match, skip entire subtree
- [ ] Unit tests: Build, diff with known trees, subtree skip verification

### Task 7.3: Sync Engine (`internal/sync/engine.go`)
```
Priority: P0 | Effort: XL | Dependencies: 7.1, 7.2, 2.5, 5.1
```
- [ ] `Negotiate(deviceID, clientMerkleRoot)`:
  - Build server Merkle tree for user
  - Compare roots → if equal, return "synced"
  - Walk tree, compute diff
  - Return: diff (added/modified/deleted), server vector clock
- [ ] `Push(deviceID, chunks[])`:
  - Receive chunks from client
  - Store new chunks, dedup existing
  - Track received chunks per session
- [ ] `Pull(deviceID, chunkHashes[])`:
  - Stream requested chunks to client
  - Bandwidth tracking
- [ ] `Commit(deviceID, vectorClock, fileUpdates[])`:
  - Atomic: Update VFS metadata, create manifests, update sync state
  - Conflict check: Compare vector clocks for each file
  - If conflict → invoke ConflictResolver
  - Broadcast changes to other devices via WebSocket
  - Return: new Merkle root, resolved conflicts
- [ ] Unit tests: Full sync cycle, multi-device scenarios, network interruption recovery

### Task 7.4: Conflict Resolution (`internal/sync/conflict.go`)
```
Priority: P0 | Effort: M | Dependencies: 7.3
```
- [ ] `Detect(localClock, remoteClock)` — Vector clock concurrent check
- [ ] `AutoResolve(conflict)`:
  - Non-overlapping chunk changes → merge manifests
  - Overlapping changes → conflict copy strategy
  - Conflict copy naming: `file.conflict-{device}-{timestamp}.ext`
- [ ] `ManualResolve(conflictID, strategy)` — Accept local/remote/merged version
- [ ] Conflict dashboard data: List unresolved, resolution history
- [ ] Unit tests: All conflict scenarios, auto-merge, conflict copy naming

### Task 7.5: Sync API Handlers (`internal/api/sync.go`)
```
Priority: P0 | Effort: M | Dependencies: 7.3
```
- [ ] `POST /api/v1/sync/negotiate` — Wire up to sync engine
- [ ] `POST /api/v1/sync/push` — Chunked upload for sync
- [ ] `GET /api/v1/sync/pull` — Chunked download for sync
- [ ] `POST /api/v1/sync/commit` — Atomic commit
- [ ] `GET /api/v1/sync/status` — Current sync state for device
- [ ] Unit tests: API layer tests

### Task 7.6: WebSocket Change Feed (`internal/api/websocket.go`, `internal/notify/`)
```
Priority: P0 | Effort: M | Dependencies: 5.2
```
- [ ] `hub.go` — WebSocket hub: Register/unregister clients, fan-out events per user
- [ ] `WS /api/v1/sync/ws` — WebSocket upgrade handler:
  - Auth: JWT token in query param or first message
  - Device ID registration
  - Server → Client events: file.created, file.updated, file.deleted, sync.complete, conflict.detected, share.received
  - Client → Server: ping/pong keepalive
- [ ] Reconnection handling: Client missed events → full sync on reconnect
- [ ] Unit tests: Hub fan-out, auth, event types

---

## Phase 8 — WebDAV Server

### Task 8.1: WebDAV Core (`internal/webdav/`)
```
Priority: P0 | Effort: XL | Dependencies: 5.1, 3.4, 4.7
```
- [ ] `server.go` — Main handler, method dispatch
- [ ] `xml.go` — WebDAV XML marshaling/unmarshaling:
  - DAV: namespace handling
  - Multistatus response builder
  - Property value types (date, int, string, href)
- [ ] `propfind.go`:
  - Depth: 0 (file only), 1 (dir + children), infinity (recursive, limited)
  - Properties: displayname, getcontentlength, getcontenttype, getlastmodified, getetag, resourcetype, supportedlock, lockdiscovery
  - Allprop / propname modes
- [ ] `proppatch.go` — Set/remove custom properties (stored in DB)
- [ ] `methods.go`:
  - `GET` — File download (stream from chunks, decrypt if E2E)
  - `PUT` — File upload (chunk, store, create VFS entry)
  - `DELETE` — Soft delete
  - `MKCOL` — Create directory
  - `COPY` — Copy file/folder (Depth: 0 or infinity)
  - `MOVE` — Rename/move (Depth: infinity)
- [ ] `lock.go` — Class 2 locking:
  - LOCK: Exclusive/shared write lock, timeout, lock token
  - UNLOCK: Release lock by token
  - If-header parsing for conditional requests
  - Lock timeout: Default 30m, configurable
  - In-memory lock store (refreshable)
- [ ] Endpoint: `/dav/{username}/` — Auth via HTTP Basic (password = API token)
- [ ] E2E transparent: WebDAV operations encrypt/decrypt seamlessly
- [ ] Unit tests: Compliance test suite (litmus test patterns), each method, locking scenarios

---

## Phase 9 — Web UI (React 19 + Tailwind 4.1 + Shadcn + Lucide)

### Task 9.0: React Project Setup (`web/`)
```
Priority: P0 | Effort: M | Dependencies: None
```
- [ ] Vite 6 + React 19 + TypeScript project scaffolding in `web/`
- [ ] Tailwind CSS 4.1 installation (new @theme syntax, CSS-first config)
- [ ] Shadcn UI init (New York style, CSS variables)
- [ ] Lucide React icons installation
- [ ] Dark/Light theme system:
  - CSS variables for color management
  - `ThemeProvider` component (localStorage persist + system preference)
  - Tailwind `dark:` variant support
- [ ] Responsive breakpoints: mobile-first (sm: 640, md: 768, lg: 1024, xl: 1280)
- [ ] API client module (`web/src/lib/api.ts`):
  - Fetch wrapper with JWT injection
  - Auto-refresh on 401
  - Type-safe API functions
- [ ] WebSocket client (`web/src/lib/ws.ts`):
  - Auto-reconnect with exponential backoff
  - Event emitter pattern
- [ ] Router setup (React Router v7 or TanStack Router)
- [ ] Zustand state management setup
- [ ] Build script: `npm run build` → `web/dist/` → `go:embed` in `web/embed.go`
- [ ] `web/embed.go` — `//go:embed static/*` directive, SPA fallback handler

### Task 9.1: Auth Pages
```
Priority: P0 | Effort: M | Dependencies: 9.0, 5.6
```
- [ ] **Login Page**:
  - Shadcn Card + Form (username, password)
  - "Remember me" toggle
  - Error toast on failure
  - TOTP dialog (appears after successful password if TOTP enabled)
  - Responsive: Full-width card on mobile, centered on desktop
- [ ] **First-time Setup Page** (shown if no admin exists):
  - Admin account creation
  - Server URL config
  - Storage backend selection
- [ ] **Session Management**: Token store in memory (not localStorage), refresh interceptor
- [ ] Dark/Light toggle on login page

### Task 9.2: Layout Shell & Navigation
```
Priority: P0 | Effort: M | Dependencies: 9.0
```
- [ ] **App Shell**: Sidebar + Header + Main content area
- [ ] **Sidebar** (desktop):
  - Logo + app name
  - Navigation: Files, Shared with me, Shared by me, Trash, Settings
  - Admin section (conditional): Users, System, Audit log
  - Storage quota indicator (progress bar)
  - Collapse to icons on narrow screens
- [ ] **Mobile**: Bottom tab navigation or hamburger drawer
- [ ] **Header**:
  - Breadcrumb navigation (current path)
  - Search bar (Shadcn Command / search input)
  - Sync status indicator (icon + tooltip)
  - Notifications dropdown (Shadcn Popover)
  - User avatar + dropdown menu (profile, settings, logout)
  - Theme toggle (sun/moon icon)
- [ ] Lucide icons throughout

### Task 9.3: File Manager Component
```
Priority: P0 | Effort: XL | Dependencies: 9.2, 5.3
```
- [ ] **Dual View Toggle**: Grid view (thumbnail cards) / List view (Shadcn Table)
- [ ] **Grid View**:
  - Thumbnail preview for images/videos
  - File type icon (Lucide) for documents
  - File name, size, modified date below
  - Hover overlay with quick actions
- [ ] **List View**:
  - Sortable columns: Name, Size, Modified, Type
  - Row selection (checkbox)
  - Inline actions column
- [ ] **Breadcrumb**: Clickable path segments, folder quick-jump dropdown
- [ ] **Multi-select**: Click (single), Ctrl+Click (add), Shift+Click (range)
- [ ] **Bulk Actions Toolbar** (appears on selection):
  - Download, Move, Copy, Delete, Share buttons
  - Selected count badge
- [ ] **Context Menu** (right-click / long-press on mobile):
  - Open, Download, Rename, Move to, Copy to, Share, Delete
  - Shadcn DropdownMenu
- [ ] **Inline Rename**: Double-click → editable input → Enter/Escape
- [ ] **Empty State**: Illustration + "No files yet" + upload button
- [ ] **Loading**: Shadcn Skeleton rows/cards while fetching
- [ ] Responsive: Grid auto-columns (1 col mobile → 4-6 cols desktop), list hides columns on mobile

### Task 9.4: Upload Manager Component
```
Priority: P0 | Effort: L | Dependencies: 9.3, 5.4
```
- [ ] **Drag & Drop Zone**: Full-page drop overlay, highlighted border
- [ ] **File Picker**: Button + hidden input (multiple files + folder upload)
- [ ] **Upload Queue Panel** (bottom sheet / slide-over):
  - Per-file progress bar (Shadcn Progress)
  - File name, size, speed, ETA
  - Pause/Resume/Cancel per file
  - Overall progress summary
- [ ] **Chunked Upload Logic**:
  - Split file using client-side CDC (Web Worker for non-blocking)
  - Upload chunks in parallel (4 concurrent)
  - Retry failed chunks
  - Dedup: Send chunk hash first, skip if server has it
- [ ] **E2E Encryption**: Encrypt chunks in Web Worker before upload
- [ ] **Upload Complete**: Toast notification, file appears in list (optimistic UI)
- [ ] Responsive: Bottom sheet on mobile, side panel on desktop

### Task 9.5: File Preview Panel
```
Priority: P0 | Effort: M | Dependencies: 9.3, 5.5
```
- [ ] **Side Panel / Modal**: Opens on file click (single-click list, double-click grid)
- [ ] **Image Preview**: `<img>` with zoom/pan
- [ ] **Video Preview**: `<video>` player (native HTML5)
- [ ] **Audio Preview**: `<audio>` player
- [ ] **PDF Preview**: Embedded `<iframe>` or PDF.js
- [ ] **Text/Code Preview**: Syntax highlighted (lightweight, not CodeMirror)
- [ ] **Markdown Preview**: Rendered HTML
- [ ] **Other files**: Type icon + metadata + download button
- [ ] **Info Tab**: Full metadata (size, type, created, modified, version, path, checksum)
- [ ] **Versions Tab**: Version history list, download/restore specific version
- [ ] **Sharing Tab**: Current shares, quick share actions
- [ ] Responsive: Full-screen modal on mobile, side panel on desktop

### Task 9.6: Share Dialog & Manager
```
Priority: P0 | Effort: L | Dependencies: 9.3, 6.1, 6.2
```
- [ ] **Create Share Dialog** (Shadcn Dialog):
  - Tab 1 — Link Share:
    - Toggle: Enable public link
    - Password field (optional)
    - Expiry date picker (Shadcn DatePicker)
    - Max downloads input
    - Preview-only toggle
    - Copy link button, QR code display
  - Tab 2 — User Share:
    - User search autocomplete (Shadcn Combobox)
    - Permission selector (read / write / manage)
    - Share button
    - List of current user shares with remove option
- [ ] **Shared by Me Page**: Table of all active shares with stats (views, downloads)
- [ ] **Shared with Me Page**: List of files shared by others, permission badges
- [ ] **Public Share View** (`/s/{token}`):
  - Clean landing page with file info
  - Password gate (if protected)
  - Download button, preview (if allowed)
  - Folder browse mode
  - VaultDrift branding footer
- [ ] Responsive: Dialog → full-screen on mobile

### Task 9.7: Sync Dashboard Component
```
Priority: P1 | Effort: M | Dependencies: 9.2, 7.5, 7.6
```
- [ ] **Sync Status Overview**:
  - Current state: Synced / Syncing / Paused / Error
  - Last sync timestamp
  - Connected devices list
- [ ] **Device List** (Shadcn Table):
  - Device name, type icon, OS, last sync, status
  - Remove device action
- [ ] **Conflict Queue**:
  - List of unresolved conflicts
  - Per-conflict: File name, device names, timestamps
  - Actions: Keep local, Keep remote, Keep both
  - Side-by-side diff viewer (for text files)
- [ ] **Activity Feed**: Real-time sync events (WebSocket powered)
- [ ] Responsive: Stacked cards on mobile

### Task 9.8: Admin Panel
```
Priority: P1 | Effort: L | Dependencies: 9.2, 5.7, 5.8
```
- [ ] **User Management Page**:
  - User table (Shadcn DataTable): Name, email, role, quota usage, status, last login
  - Create user dialog
  - Edit user dialog (role, quota, status)
  - Delete user confirmation
  - Quota visualization (progress bar per user)
- [ ] **System Stats Dashboard**:
  - Cards: Total storage, users, files, dedup savings
  - Storage usage chart (Recharts bar/pie chart)
  - Active connections count
- [ ] **Audit Log Viewer**:
  - Filterable table: User, action, resource, date range
  - Search within details
  - Export to CSV
- [ ] **Settings Page**:
  - Storage backend config
  - SMTP settings + test button
  - Security settings (registration, TOTP enforcement, lockout duration)
  - Branding (custom logo, app name)
- [ ] Responsive: Tables scroll horizontally on mobile, cards stack

### Task 9.9: User Settings Page
```
Priority: P1 | Effort: M | Dependencies: 9.2, 5.6, 5.7
```
- [ ] **Profile Section**: Avatar upload, display name, email edit
- [ ] **Security Section**:
  - Change password form
  - TOTP setup: QR code display (Shadcn Dialog), verification step, backup codes
  - Recovery key: Generate, download, warning dialog
- [ ] **API Tokens Section**:
  - List tokens (name, created, last used, permissions)
  - Generate new token dialog (show once, copy)
  - Revoke token
- [ ] **Connected Devices Section**:
  - Device list with last active
  - Rename device
  - Revoke device
- [ ] **Appearance Section**:
  - Theme: Light / Dark / System
  - Language (future)
- [ ] Responsive: Section cards stack on mobile

### Task 9.10: Embedded Build Pipeline
```
Priority: P0 | Effort: M | Dependencies: 9.0
```
- [ ] `web/package.json` — Vite build config, output to `web/dist/`
- [ ] `web/embed.go`:
  ```go
  //go:embed dist/*
  var StaticFS embed.FS
  ```
- [ ] SPA fallback: Serve `index.html` for all non-API/non-DAV routes
- [ ] Cache headers: Hashed filenames (Vite default) → `Cache-Control: max-age=31536000`
- [ ] `index.html` → `Cache-Control: no-cache` (always fresh)
- [ ] Gzip pre-compression: Optionally serve `.gz` files
- [ ] Makefile integration: `make build-web` → `cd web && npm run build`
- [ ] `make build` depends on `build-web`

---

## Phase 10 — CLI Client

### Task 10.1: CLI Framework (`cmd/vaultdrift-cli/`)
```
Priority: P0 | Effort: M | Dependencies: None
```
- [ ] CLI argument parser (custom, no cobra/urfave — zero deps):
  - Command hierarchy: `vaultdrift-cli <command> [subcommand] [flags]`
  - Flag parsing: `--key value`, `--key=value`, `-k value`, boolean flags
  - Help generation: `--help` per command
  - Version: `--version`
- [ ] Config file: `~/.vaultdrift/config.yaml` (server URL, auth token, sync folder, device ID)
- [ ] Color output: ANSI escape codes, auto-detect terminal

### Task 10.2: API Client Library (`client/client.go`)
```
Priority: P0 | Effort: M | Dependencies: 5.2
```
- [ ] `Client` struct: Base URL, auth token, HTTP client
- [ ] Methods mirroring all REST API endpoints (type-safe)
- [ ] Auto-refresh: 401 → refresh token → retry
- [ ] WebSocket connection management
- [ ] Chunk upload/download with progress callbacks
- [ ] Shared between CLI and Desktop (same `client/` package)

### Task 10.3: CLI Commands
```
Priority: P0 | Effort: L | Dependencies: 10.1, 10.2
```
- [ ] `init` — Interactive setup: server URL, login, choose sync folder, register device
- [ ] `login` — Authenticate, store tokens in config
- [ ] `logout` — Clear stored tokens
- [ ] `sync` — One-time sync cycle (negotiate → transfer → commit)
- [ ] `ls [path]` — List remote directory (table format)
- [ ] `upload <local> <remote>` — Upload file with progress bar
- [ ] `download <remote> [local]` — Download file with progress bar
- [ ] `mkdir <path>` — Create remote directory
- [ ] `rm <path>` — Delete remote file/folder
- [ ] `mv <src> <dst>` — Move/rename
- [ ] `share <path>` — Create share link, print URL
- [ ] `config get/set <key> <value>` — View/update config
- [ ] `conflicts` — List unresolved conflicts
- [ ] `conflicts resolve <id> --strategy=<local|remote|both>` — Resolve conflict

### Task 10.4: Sync Daemon (`client/daemon.go`)
```
Priority: P0 | Effort: XL | Dependencies: 10.2, 7.3
```
- [ ] `daemon start` — Background process (fork or `nohup`)
- [ ] `daemon stop` — Send SIGTERM to daemon PID
- [ ] `daemon status` — Print sync state, last sync time, pending changes
- [ ] Filesystem watcher (`client/watcher/`):
  - `watcher_linux.go` — inotify via syscall
  - `watcher_darwin.go` — FSEvents via CGo-free approach (or kqueue)
  - `watcher_windows.go` — ReadDirectoryChangesW via syscall
  - Debounce: 500ms after last change before triggering sync
  - Ignore patterns: `.vaultdrift/`, `.git/`, temp files
- [ ] Sync loop:
  1. Watch for local changes → queue
  2. Debounce → batch changes
  3. Build local Merkle tree
  4. Negotiate with server
  5. Push local changes
  6. Pull remote changes
  7. Apply remote changes to local FS
  8. Commit
  9. Update local state
- [ ] Selective sync: Choose which remote folders to sync locally
- [ ] Bandwidth throttling: `--max-upload`, `--max-download` (bytes/sec)
- [ ] Retry: Exponential backoff on network failure (1s → 2s → 4s → ... → 5m max)
- [ ] WebSocket listener: Instant sync trigger on server event
- [ ] Poll fallback: Every 30s if WebSocket disconnected
- [ ] PID file: `~/.vaultdrift/daemon.pid`
- [ ] Log file: `~/.vaultdrift/daemon.log`
- [ ] Unit tests: Watcher events, sync loop states, conflict handling

---

## Phase 11 — Desktop Tray App

### Task 11.1: System Tray Integration (`desktop/`)
```
Priority: P2 | Effort: L | Dependencies: 10.4
```
- [ ] `tray_linux.go` — D-Bus StatusNotifierItem / AppIndicator (pure Go, no CGo)
- [ ] `tray_darwin.go` — macOS NSStatusBar (CGo-free via command bridge or minimal CGo)
- [ ] `tray_windows.go` — Shell_NotifyIcon via syscall
- [ ] `tray.go` — Platform-agnostic interface
- [ ] `menu.go` — Menu builder:
  - Status item (Synced / Syncing / Conflicts)
  - Open Web UI → launch browser
  - Open Sync Folder → launch file manager
  - Separator
  - Recent Activity submenu
  - Sync Now → force immediate sync
  - Pause / Resume Sync
  - Separator
  - Preferences → future settings window
  - Resolve Conflicts → open Web UI conflicts page
  - Separator
  - Quit
- [ ] `notification.go` — Desktop notifications:
  - Linux: D-Bus notifications
  - macOS: NSUserNotification (or `osascript` bridge)
  - Windows: Toast notifications via syscall
- [ ] Tray icon: Animated sync icon, static synced icon, warning icon (conflict)
- [ ] Build: `cmd/vaultdrift-desktop/main.go` → starts daemon + tray
- [ ] Same sync daemon code from Phase 10, just adds tray UI on top

---

## Phase 12 — TLS & Security Hardening

### Task 12.1: TLS & ACME (`internal/tls/`)
```
Priority: P1 | Effort: M | Dependencies: 5.2
```
- [ ] `tls.go` — TLS config: TLS 1.2+ (prefer 1.3), secure cipher suites, HSTS
- [ ] `acme.go` — Let's Encrypt ACME client:
  - HTTP-01 challenge handler
  - Certificate auto-renewal (30 days before expiry)
  - Certificate storage in `{data_dir}/certs/`
  - Pure Go ACME implementation (no certbot dependency)
- [ ] Manual cert support: Load PEM cert + key from config paths
- [ ] Self-signed cert generation for development

### Task 12.2: Security Middleware & Headers
```
Priority: P1 | Effort: S | Dependencies: 5.2
```
- [ ] Rate limiter: Token bucket per-IP + per-user, configurable limits
- [ ] CSRF: SameSite=Strict cookies + `X-Requested-With` header check for API
- [ ] Security headers:
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: strict-origin-when-cross-origin`
- [ ] Input validation: Max request body (100MB), path traversal checks

---

## Phase 13 — Background Workers & Maintenance

### Task 13.1: Chunk Garbage Collector
```
Priority: P1 | Effort: M | Dependencies: 1.4, 2.2
```
- [ ] Run hourly (configurable)
- [ ] Find chunks with ref_count = 0
- [ ] Grace period: Only delete if orphaned for > 24h
- [ ] Delete from storage backend, then from DB
- [ ] Log: Chunks deleted, bytes reclaimed

### Task 13.2: Session & Token Cleanup
```
Priority: P1 | Effort: S | Dependencies: 1.6
```
- [ ] Run every 15 minutes
- [ ] Delete expired sessions
- [ ] Delete expired API tokens
- [ ] Log cleanup stats

### Task 13.3: Trash Auto-Purge
```
Priority: P1 | Effort: S | Dependencies: 5.1
```
- [ ] Run daily
- [ ] Configurable retention: default 30 days
- [ ] Permanently delete files in trash older than retention
- [ ] Decrement chunk ref counts, trigger GC if needed

### Task 13.4: Thumbnail Generator (`internal/thumbnail/`)
```
Priority: P2 | Effort: M | Dependencies: 2.2, 5.5
```
- [ ] `image.go` — Pure Go image resize (Lanczos or bilinear):
  - Input: JPEG, PNG, WebP, GIF
  - Output: WebP thumbnail (150x150 fit)
  - Use Go's `image` stdlib + `image/jpeg`, `image/png`
- [ ] On-demand queue: Generate on first thumbnail request
- [ ] Cache: Store in `{dataDir}/thumbnails/{userID}/{fileHash}.webp`
- [ ] Invalidation: Delete cached thumbnail on file update
- [ ] Background worker: Pre-generate for recently uploaded images

### Task 13.5: SMTP Integration (`internal/notify/smtp.go`)
```
Priority: P2 | Effort: M | Dependencies: 0.2
```
- [ ] Pure Go SMTP client (no external dep):
  - STARTTLS support
  - PLAIN / LOGIN authentication
  - MIME email construction (text + HTML multipart)
- [ ] Notification templates:
  - Share invitation: "User X shared 'File Y' with you"
  - Storage quota warning: "You're using 90% of your storage"
  - Password reset (future)
- [ ] Admin test button: Send test email from admin panel

---

## Phase 14 — Testing & Quality

### Task 14.1: Unit Test Suite
```
Priority: P0 (ongoing) | Effort: XL | Dependencies: All phases
```
- [ ] Every package: `*_test.go` with 80%+ coverage
- [ ] Table-driven tests for all edge cases
- [ ] Mock interfaces: Storage backend, DB, HTTP client
- [ ] `go test -race` — No race conditions

### Task 14.2: Integration Tests
```
Priority: P1 | Effort: L | Dependencies: Phase 5+
```
- [ ] Full API test suite: Start server → run all endpoints → verify
- [ ] WebDAV compliance: Litmus test patterns
- [ ] Multi-client sync: Simulate 2-3 clients syncing concurrently
- [ ] S3 integration: MinIO container test (`-tags=integration`)
- [ ] Build tag: `//go:build integration`

### Task 14.3: Benchmark Tests
```
Priority: P1 | Effort: M | Dependencies: Phase 2, 3
```
- [ ] Chunking throughput: `BenchmarkChunk` (target: >500MB/s)
- [ ] Encryption throughput: `BenchmarkEncryptChunk` (target: >300MB/s)
- [ ] Merkle tree build: `BenchmarkMerkleBuild` (10K files)
- [ ] File listing: `BenchmarkListDir` (10K entries)
- [ ] Upload pipeline: `BenchmarkUpload` (100MB file end-to-end)

### Task 14.4: Fuzz Tests
```
Priority: P2 | Effort: M | Dependencies: Phase 5, 8
```
- [ ] `FuzzPathNormalize` — Path sanitization
- [ ] `FuzzAPIInput` — JSON request parsing
- [ ] `FuzzWebDAVXML` — XML request parsing
- [ ] `FuzzChunker` — CDC with random data

---

## Phase 15 — Documentation & Release

### Task 15.1: Documentation
```
Priority: P1 | Effort: L | Dependencies: All phases
```
- [ ] `README.md` — Complete with badges, screenshots, quick start, features
- [ ] `docs/api.md` — Full REST API documentation
- [ ] `docs/webdav.md` — WebDAV setup guide (macOS, Windows, Linux clients)
- [ ] `docs/sync.md` — Sync protocol technical documentation
- [ ] `docs/encryption.md` — E2E encryption architecture, key management
- [ ] `docs/cli.md` — CLI reference with examples
- [ ] `docs/admin.md` — Administration guide
- [ ] `docs/docker.md` — Docker deployment guide
- [ ] `vaultdrift.yaml.example` — Annotated config with all options

### Task 15.2: Release Pipeline
```
Priority: P1 | Effort: M | Dependencies: 0.1
```
- [ ] GitHub Actions CI:
  - Test on push (Linux, macOS, Windows)
  - Lint (go vet, staticcheck)
  - Build all platforms
  - Integration tests (with MinIO service)
- [ ] Release: Tag → build all binaries → create GitHub release
- [ ] Docker: Build + push to ghcr.io/vaultdrift/vaultdrift
- [ ] Install script: `curl -fsSL https://vaultdrift.com/install.sh | sh`

### Task 15.3: BRANDING.md
```
Priority: P2 | Effort: S | Dependencies: None
```
- [ ] Project name, tagline, domain, logo brief
- [ ] Color palette, typography
- [ ] Social media copy
- [ ] Nano Banana 2 prompt (logo/infographic)

---

## Task Summary

| Phase | Tasks | Priority | Estimated Effort |
|-------|-------|----------|-----------------|
| 0 — Scaffolding | 3 tasks | P0 | S + M + S |
| 1 — Data Layer | 6 tasks | P0 | L + M + M + S + S + M |
| 2 — Storage & Chunking | 5 tasks | P0 | S + M + XL + M + L |
| 3 — Encryption | 4 tasks | P0-P1 | L + M + M + M |
| 4 — Auth & RBAC | 7 tasks | P0-P1 | S + M + M + M + M + S + M |
| 5 — VFS & Core API | 8 tasks | P0-P1 | L + M + L + L + M + M + M + M |
| 6 — Sharing | 2 tasks | P0 | L + M |
| 7 — Sync Protocol | 6 tasks | P0 | S + M + XL + M + M + M |
| 8 — WebDAV | 1 task | P0 | XL |
| 9 — Web UI (React) | 11 tasks | P0-P1 | M + M + M + XL + L + M + L + L + M + L + M |
| 10 — CLI Client | 4 tasks | P0 | M + M + L + XL |
| 11 — Desktop Tray | 1 task | P2 | L |
| 12 — TLS & Security | 2 tasks | P1 | M + S |
| 13 — Background Workers | 5 tasks | P1-P2 | M + S + S + M + M |
| 14 — Testing | 4 tasks | P0-P2 | XL + L + M + M |
| 15 — Docs & Release | 3 tasks | P1-P2 | L + M + S |

**Total: 72 tasks**

---

## Suggested Build Order

```
Phase 0 (Scaffolding)
  ↓
Phase 1 (CobaltDB + Data Layer)
  ↓
Phase 2 (Storage + Chunking) ←→ Phase 3 (Encryption) [parallel]
  ↓
Phase 4 (Auth/RBAC)
  ↓
Phase 5 (VFS + Core API)
  ↓
Phase 6 (Sharing) ←→ Phase 7 (Sync Protocol) [parallel]
  ↓
Phase 8 (WebDAV)
  ↓
Phase 9 (Web UI) ←→ Phase 10 (CLI Client) [parallel]
  ↓
Phase 11 (Desktop Tray)
  ↓
Phase 12 (TLS) + Phase 13 (Workers) [parallel]
  ↓
Phase 14 (Testing) — ongoing throughout
  ↓
Phase 15 (Docs + Release)
```

---

*VaultDrift — 72 tasks, zero dependencies, one vision.*
