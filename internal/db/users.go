package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CreateUser creates a new user.
func (m *Manager) CreateUser(ctx context.Context, user *User) error {
	query := `INSERT INTO users (
		id, username, email, display_name, password_hash, role,
		quota_bytes, used_bytes, totp_secret, totp_enabled,
		public_key, encrypted_private_key, recovery_key_hash,
		avatar_chunk_hash, status, last_login_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Email, user.DisplayName,
		user.PasswordHash, user.Role, user.QuotaBytes, user.UsedBytes,
		user.TOTPSecret, boolToInt(user.TOTPEnabled),
		user.PublicKey, user.EncryptedPrivateKey, user.RecoveryKeyHash,
		user.AvatarChunkHash, user.Status, timePtrToStr(user.LastLoginAt),
		user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if strings.Contains(err.Error(), "username") {
				return fmt.Errorf("username already exists")
			}
			if strings.Contains(err.Error(), "email") {
				return fmt.Errorf("email already exists")
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID.
func (m *Manager) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, username, email, display_name, password_hash, role,
		quota_bytes, used_bytes, totp_secret, totp_enabled,
		public_key, encrypted_private_key, recovery_key_hash,
		avatar_chunk_hash, status, last_login_at, created_at, updated_at
	FROM users WHERE id = ?`

	user := &User{}
	var totpSecret, recoveryKeyHash, avatarChunkHash sql.NullString
	var lastLoginAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&user.PasswordHash, &user.Role, &user.QuotaBytes, &user.UsedBytes,
		&totpSecret, &user.TOTPEnabled,
		&user.PublicKey, &user.EncryptedPrivateKey, &recoveryKeyHash,
		&avatarChunkHash, &user.Status, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if totpSecret.Valid {
		user.TOTPSecret = &totpSecret.String
	}
	if recoveryKeyHash.Valid {
		user.RecoveryKeyHash = &recoveryKeyHash.String
	}
	if avatarChunkHash.Valid {
		user.AvatarChunkHash = &avatarChunkHash.String
	}
	if lastLoginAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
		user.LastLoginAt = &t
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (m *Manager) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, email, display_name, password_hash, role,
		quota_bytes, used_bytes, totp_secret, totp_enabled,
		public_key, encrypted_private_key, recovery_key_hash,
		avatar_chunk_hash, status, last_login_at, created_at, updated_at
	FROM users WHERE username = ?`

	user := &User{}
	var totpSecret, recoveryKeyHash, avatarChunkHash sql.NullString
	var lastLoginAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&user.PasswordHash, &user.Role, &user.QuotaBytes, &user.UsedBytes,
		&totpSecret, &user.TOTPEnabled,
		&user.PublicKey, &user.EncryptedPrivateKey, &recoveryKeyHash,
		&avatarChunkHash, &user.Status, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if totpSecret.Valid {
		user.TOTPSecret = &totpSecret.String
	}
	if recoveryKeyHash.Valid {
		user.RecoveryKeyHash = &recoveryKeyHash.String
	}
	if avatarChunkHash.Valid {
		user.AvatarChunkHash = &avatarChunkHash.String
	}
	if lastLoginAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
		user.LastLoginAt = &t
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (m *Manager) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, username, email, display_name, password_hash, role,
		quota_bytes, used_bytes, totp_secret, totp_enabled,
		public_key, encrypted_private_key, recovery_key_hash,
		avatar_chunk_hash, status, last_login_at, created_at, updated_at
	FROM users WHERE email = ?`

	user := &User{}
	var totpSecret, recoveryKeyHash, avatarChunkHash sql.NullString
	var lastLoginAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&user.PasswordHash, &user.Role, &user.QuotaBytes, &user.UsedBytes,
		&totpSecret, &user.TOTPEnabled,
		&user.PublicKey, &user.EncryptedPrivateKey, &recoveryKeyHash,
		&avatarChunkHash, &user.Status, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if totpSecret.Valid {
		user.TOTPSecret = &totpSecret.String
	}
	if recoveryKeyHash.Valid {
		user.RecoveryKeyHash = &recoveryKeyHash.String
	}
	if avatarChunkHash.Valid {
		user.AvatarChunkHash = &avatarChunkHash.String
	}
	if lastLoginAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
		user.LastLoginAt = &t
	}

	return user, nil
}

// UpdateUser updates a user's fields.
func (m *Manager) UpdateUser(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	allowedFields := map[string]bool{
		"username":              true,
		"email":                 true,
		"display_name":          true,
		"password_hash":         true,
		"role":                  true,
		"quota_bytes":           true,
		"used_bytes":            true,
		"totp_secret":           true,
		"totp_enabled":          true,
		"public_key":            true,
		"encrypted_private_key": true,
		"recovery_key_hash":     true,
		"avatar_chunk_hash":     true,
		"status":                true,
		"last_login_at":         true,
	}

	setClauses := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+1)

	for field, value := range updates {
		if !allowedFields[field] {
			return fmt.Errorf("invalid update field: %s", field)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
		args = append(args, value)
	}

	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s, updated_at = ? WHERE id = ?",
		strings.Join(setClauses, ", "))
	args = append([]any{time.Now().UTC().Format(time.RFC3339)}, args...)

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeleteUser deletes a user and all associated data.
func (m *Manager) DeleteUser(ctx context.Context, id string) error {
	// Due to CASCADE, this will also delete sessions, files, devices, etc.
	result, err := m.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ListUsers retrieves a paginated list of users with optional filtering.
func (m *Manager) ListUsers(ctx context.Context, offset, limit int, filter UserFilter) ([]*User, int, error) {
	whereClauses := []string{"1 = 1"}
	args := []any{}

	if filter.Role != "" {
		whereClauses = append(whereClauses, "role = ?")
		args = append(args, filter.Role)
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filter.Status)
	}

	if filter.Query != "" {
		whereClauses = append(whereClauses, "(username LIKE ? OR email LIKE ? OR display_name LIKE ?)")
		like := "%" + filter.Query + "%"
		args = append(args, like, like, like)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM users WHERE " + strings.Join(whereClauses, " AND ")
	var total int
	if err := m.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	query := `SELECT id, username, email, display_name, password_hash, role,
		quota_bytes, used_bytes, totp_secret, totp_enabled,
		public_key, encrypted_private_key, recovery_key_hash,
		avatar_chunk_hash, status, last_login_at, created_at, updated_at
	FROM users WHERE ` + strings.Join(whereClauses, " AND ") +
		` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	args = append(args, limit, offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]*User, 0)
	for rows.Next() {
		user := &User{}
		var totpSecret, recoveryKeyHash, avatarChunkHash sql.NullString
		var lastLoginAt sql.NullString

		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.DisplayName,
			&user.PasswordHash, &user.Role, &user.QuotaBytes, &user.UsedBytes,
			&totpSecret, &user.TOTPEnabled,
			&user.PublicKey, &user.EncryptedPrivateKey, &recoveryKeyHash,
			&avatarChunkHash, &user.Status, &lastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		if totpSecret.Valid {
			user.TOTPSecret = &totpSecret.String
		}
		if recoveryKeyHash.Valid {
			user.RecoveryKeyHash = &recoveryKeyHash.String
		}
		if avatarChunkHash.Valid {
			user.AvatarChunkHash = &avatarChunkHash.String
		}
		if lastLoginAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
			user.LastLoginAt = &t
		}

		users = append(users, user)
	}

	return users, total, nil
}

// UpdateUsedBytes atomically updates a user's used storage quota.
func (m *Manager) UpdateUsedBytes(ctx context.Context, id string, delta int64) error {
	query := "UPDATE users SET used_bytes = used_bytes + ?, updated_at = ? WHERE id = ?"
	result, err := m.db.ExecContext(ctx, query, delta, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update used bytes: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetUserCount returns the total number of users.
func (m *Manager) GetUserCount(ctx context.Context) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}

// GetUserCountByStatus returns the number of users by status.
func (m *Manager) GetUserCountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE status = ?", status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}

// Helper functions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func timePtrToStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
