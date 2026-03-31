package db

import (
	"time"
)

// User represents a user account.
type User struct {
	ID                  string     `json:"id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	DisplayName         string     `json:"display_name"`
	PasswordHash        string     `json:"-"`
	Role                string     `json:"role"`
	QuotaBytes          int64      `json:"quota_bytes"`
	UsedBytes           int64      `json:"used_bytes"`
	TOTPSecret          *string    `json:"-"`
	TOTPEnabled         bool       `json:"totp_enabled"`
	PublicKey           []byte     `json:"-"`
	EncryptedPrivateKey []byte     `json:"-"`
	RecoveryKeyHash     *string    `json:"-"`
	AvatarChunkHash     *string    `json:"-"`
	Status              string     `json:"status"`
	LastLoginAt         *time.Time `json:"last_login_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// Session represents an active user session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	RefreshToken string    `json:"-"`
	DeviceName   string    `json:"device_name"`
	DeviceType   string    `json:"device_type"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// APIToken represents an API access token.
type APIToken struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	TokenHash   string     `json:"-"`
	Permissions []string   `json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// File represents a file or folder in the virtual filesystem.
type File struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	ParentID      *string    `json:"parent_id"`
	Name          string     `json:"name"`
	NameEncrypted []byte     `json:"-"`
	Type          string     `json:"type"` // "file" or "folder"
	SizeBytes     int64      `json:"size_bytes"`
	MimeType      string     `json:"mime_type"`
	ManifestID    *string    `json:"manifest_id"`
	Checksum      *string    `json:"checksum"`
	IsEncrypted   bool       `json:"is_encrypted"`
	EncryptedKey  []byte     `json:"-"`
	IsTrashed     bool       `json:"is_trashed"`
	TrashedAt     *time.Time `json:"trashed_at"`
	Version       int        `json:"version"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Manifest represents a file version manifest.
type Manifest struct {
	ID         string    `json:"id"`
	FileID     string    `json:"file_id"`
	Version    int       `json:"version"`
	SizeBytes  int64     `json:"size_bytes"`
	ChunkCount int       `json:"chunk_count"`
	Chunks     []string  `json:"chunks"`
	Checksum   string    `json:"checksum"`
	DeviceID   string    `json:"device_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// Chunk represents a content-addressed chunk.
type Chunk struct {
	Hash           string    `json:"hash"`
	SizeBytes      int64     `json:"size_bytes"`
	StorageBackend string    `json:"storage_backend"`
	StoragePath    string    `json:"storage_path"`
	RefCount       int       `json:"ref_count"`
	IsEncrypted    bool      `json:"is_encrypted"`
	CreatedAt      time.Time `json:"created_at"`
}

// Share represents a share link or user share.
type Share struct {
	ID            string     `json:"id"`
	FileID        string     `json:"file_id"`
	CreatedBy     string     `json:"created_by"`
	ShareType     string     `json:"share_type"` // "link" or "user"
	Token         *string    `json:"token,omitempty"`
	PasswordHash  *string    `json:"-"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	MaxDownloads  *int       `json:"max_downloads,omitempty"`
	DownloadCount int        `json:"download_count"`
	AllowUpload   bool       `json:"allow_upload"`
	PreviewOnly   bool       `json:"preview_only"`
	SharedWith    *string    `json:"shared_with,omitempty"`
	Permission    string     `json:"permission"`
	EncryptedKey  []byte     `json:"-"`
	IsActive      bool       `json:"is_active"`
	ViewCount     int        `json:"view_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Device represents a synced device.
type Device struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	DeviceType  string     `json:"device_type"`
	OS          string     `json:"os"`
	SyncFolder  string     `json:"sync_folder"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
	VectorClock string     `json:"vector_clock"`
	MerkleRoot  *string    `json:"merkle_root"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SyncState represents the sync state for a file on a device.
type SyncState struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	FileID      string    `json:"file_id"`
	ManifestID  string    `json:"manifest_id"`
	VectorClock string    `json:"vector_clock"`
	SyncedAt    time.Time `json:"synced_at"`
}

// Role represents a user role.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission represents a role permission.
type Permission struct {
	ID       string `json:"id"`
	RoleID   string `json:"role_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope"`
}

// UserRole represents a user-role assignment.
type UserRole struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}

// AuditEntry represents an audit log entry.
type AuditEntry struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *string   `json:"resource_id"`
	Details      string    `json:"details"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}

// Setting represents a key-value setting.
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserFilter represents filters for listing users.
type UserFilter struct {
	Role   string
	Status string
	Query  string
}

// ListOpts represents pagination and sorting options.
type ListOpts struct {
	Offset int
	Limit  int
	Sort   string
	Order  string // "asc" or "desc"
}
