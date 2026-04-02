package db

import (
	"fmt"
)

// Schema version for migrations.
const CurrentSchemaVersion = 2

// migrate runs database migrations to bring the schema to the current version.
func (m *Manager) migrate() error {
	// Ensure schema_version table exists first
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		id INTEGER PRIMARY KEY DEFAULT 1,
		version INTEGER NOT NULL DEFAULT 0,
		applied_at TEXT NOT NULL DEFAULT ''
	)`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	currentVersion, err := m.GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	if currentVersion >= CurrentSchemaVersion {
		return nil
	}

	// Run migrations sequentially
	for version := currentVersion + 1; version <= CurrentSchemaVersion; version++ {
		if err := m.runMigration(version); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", version, err)
		}
		if err := m.SetVersion(version); err != nil {
			return fmt.Errorf("failed to set schema version %d: %w", version, err)
		}
	}

	return nil
}

// runMigration executes a specific migration version.
func (m *Manager) runMigration(version int) error {
	switch version {
	case 1:
		return m.migrationV1()
	case 2:
		return m.migrationV2()
	default:
		return fmt.Errorf("unknown migration version: %d", version)
	}
}

// migrationV1 creates the initial schema.
func (m *Manager) migrationV1() error {
	queries := []string{
		// Schema version tracking
		`CREATE TABLE IF NOT EXISTS schema_version (
			id INTEGER PRIMARY KEY DEFAULT 1,
			version INTEGER NOT NULL,
			applied_at TEXT NOT NULL
		)`,

		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			quota_bytes INTEGER NOT NULL DEFAULT 10737418240,
			used_bytes INTEGER NOT NULL DEFAULT 0,
			totp_secret TEXT DEFAULT NULL,
			totp_enabled INTEGER NOT NULL DEFAULT 0,
			public_key BLOB DEFAULT NULL,
			encrypted_private_key BLOB DEFAULT NULL,
			recovery_key_hash TEXT DEFAULT NULL,
			avatar_chunk_hash TEXT DEFAULT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			last_login_at TEXT DEFAULT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)`,

		// Sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			refresh_token TEXT UNIQUE NOT NULL,
			device_name TEXT NOT NULL DEFAULT 'Unknown',
			device_type TEXT NOT NULL DEFAULT 'web',
			ip_address TEXT NOT NULL,
			user_agent TEXT NOT NULL DEFAULT '',
			last_active_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_refresh ON sessions(refresh_token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,

		// API Tokens table
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			token_hash TEXT UNIQUE NOT NULL,
			permissions TEXT NOT NULL DEFAULT '[]',
			last_used_at TEXT DEFAULT NULL,
			expires_at TEXT DEFAULT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash)`,

		// Files table (Virtual Filesystem)
		`CREATE TABLE IF NOT EXISTS files (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			parent_id TEXT DEFAULT NULL REFERENCES files(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			name_encrypted BLOB DEFAULT NULL,
			type TEXT NOT NULL,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			manifest_id TEXT DEFAULT NULL,
			checksum TEXT DEFAULT NULL,
			is_encrypted INTEGER NOT NULL DEFAULT 0,
			encrypted_key BLOB DEFAULT NULL,
			is_trashed INTEGER NOT NULL DEFAULT 0,
			trashed_at TEXT DEFAULT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(user_id, parent_id, name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_files_user ON files(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_files_parent ON files(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_files_user_parent ON files(user_id, parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_files_name ON files(name)`,
		`CREATE INDEX IF NOT EXISTS idx_files_type ON files(type)`,
		`CREATE INDEX IF NOT EXISTS idx_files_trashed ON files(is_trashed)`,
		`CREATE INDEX IF NOT EXISTS idx_files_updated ON files(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_files_mime ON files(mime_type)`,

		// Manifests table (File Versions)
		`CREATE TABLE IF NOT EXISTS manifests (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			version INTEGER NOT NULL,
			size_bytes INTEGER NOT NULL,
			chunk_count INTEGER NOT NULL,
			chunks TEXT NOT NULL,
			checksum TEXT NOT NULL,
			device_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(file_id, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_manifests_file ON manifests(file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_manifests_file_version ON manifests(file_id, version)`,

		// Chunks table
		`CREATE TABLE IF NOT EXISTS chunks (
			hash TEXT PRIMARY KEY,
			size_bytes INTEGER NOT NULL,
			storage_backend TEXT NOT NULL,
			storage_path TEXT NOT NULL,
			ref_count INTEGER NOT NULL DEFAULT 1,
			is_encrypted INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_backend ON chunks(storage_backend)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_refcount ON chunks(ref_count)`,

		// Shares table
		`CREATE TABLE IF NOT EXISTS shares (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			share_type TEXT NOT NULL,
			token TEXT UNIQUE DEFAULT NULL,
			password_hash TEXT DEFAULT NULL,
			expires_at TEXT DEFAULT NULL,
			max_downloads INTEGER DEFAULT NULL,
			download_count INTEGER NOT NULL DEFAULT 0,
			allow_upload INTEGER NOT NULL DEFAULT 0,
			preview_only INTEGER NOT NULL DEFAULT 0,
			shared_with TEXT DEFAULT NULL REFERENCES users(id) ON DELETE CASCADE,
			permission TEXT NOT NULL DEFAULT 'read',
			encrypted_key BLOB DEFAULT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			view_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_file ON shares(file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_created_by ON shares(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_shared_with ON shares(shared_with)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_active ON shares(is_active)`,

		// Devices table (Sync)
		`CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			device_type TEXT NOT NULL,
			os TEXT NOT NULL DEFAULT '',
			sync_folder TEXT NOT NULL DEFAULT '',
			last_sync_at TEXT DEFAULT NULL,
			vector_clock TEXT NOT NULL DEFAULT '{}',
			merkle_root TEXT DEFAULT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_active ON devices(is_active)`,

		// Sync State table
		`CREATE TABLE IF NOT EXISTS sync_state (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
			file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			manifest_id TEXT NOT NULL,
			vector_clock TEXT NOT NULL,
			synced_at TEXT NOT NULL,
			UNIQUE(device_id, file_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_state_device ON sync_state(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_state_file ON sync_state(file_id)`,

		// Roles table
		`CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			is_system INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,

		// Permissions table
		`CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY,
			role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT 'own',
			UNIQUE(role_id, resource, action, scope)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_role ON permissions(role_id)`,

		// User Roles table
		`CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			PRIMARY KEY(user_id, role_id)
		)`,

		// Audit Log table
		`CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT DEFAULT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT DEFAULT NULL,
			details TEXT NOT NULL DEFAULT '{}',
			ip_address TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log(resource_type, resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at)`,

		// Settings table
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		// Federation Peers table
		`CREATE TABLE IF NOT EXISTS federation_peers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			public_url TEXT NOT NULL,
			public_key TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			last_seen TEXT NOT NULL,
			created_at TEXT NOT NULL,
			capabilities TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_federation_peers_status ON federation_peers(status)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_federation_peers_url ON federation_peers(public_url)`,

		// Federation Invites table
		`CREATE TABLE IF NOT EXISTS federation_invites (
			id TEXT PRIMARY KEY,
			from_peer_id TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at TEXT NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_federation_invites_token ON federation_invites(token)`,
		`CREATE INDEX IF NOT EXISTS idx_federation_invites_expires ON federation_invites(expires_at)`,

		// Federated Shares table
		`CREATE TABLE IF NOT EXISTS federated_shares (
			id TEXT PRIMARY KEY,
			local_file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			peer_id TEXT NOT NULL,
			remote_user_id TEXT NOT NULL,
			permission TEXT NOT NULL DEFAULT 'read',
			token TEXT UNIQUE NOT NULL,
			expires_at TEXT DEFAULT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_federated_shares_file ON federated_shares(local_file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_federated_shares_token ON federated_shares(token)`,
		`CREATE INDEX IF NOT EXISTS idx_federated_shares_peer ON federated_shares(peer_id)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w\nQuery: %s", err, query)
		}
	}

	return nil
}

// migrationV2 adds password_change_required column to users table.
func (m *Manager) migrationV2() error {
	_, err := m.db.Exec(`
		ALTER TABLE users ADD COLUMN password_change_required INTEGER NOT NULL DEFAULT 0
	`)
	if err != nil {
		// Column might already exist (if upgrading from older version)
		return nil
	}
	return nil
}
