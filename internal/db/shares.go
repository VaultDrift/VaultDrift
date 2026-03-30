package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateShare creates a new share.
func (m *Manager) CreateShare(ctx context.Context, share *Share) error {
	query := `INSERT INTO shares (id, file_id, created_by, share_type, token,
		password_hash, expires_at, max_downloads, download_count, allow_upload,
		preview_only, shared_with, permission, encrypted_key, is_active,
		view_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		share.ID, share.FileID, share.CreatedBy, share.ShareType, share.Token,
		share.PasswordHash, share.ExpiresAt, share.MaxDownloads, share.DownloadCount,
		boolToInt(share.AllowUpload), boolToInt(share.PreviewOnly),
		share.SharedWith, share.Permission, share.EncryptedKey,
		boolToInt(share.IsActive), share.ViewCount,
		share.CreatedAt.Format(time.RFC3339), share.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}

	return nil
}

// GetShareByID retrieves a share by ID.
func (m *Manager) GetShareByID(ctx context.Context, id string) (*Share, error) {
	query := `SELECT id, file_id, created_by, share_type, token, password_hash,
		expires_at, max_downloads, download_count, allow_upload, preview_only,
		shared_with, permission, encrypted_key, is_active, view_count, created_at, updated_at
	FROM shares WHERE id = ?`

	share := &Share{}
	var token, passwordHash, expiresAt, maxDownloads, sharedWith sql.NullString

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&share.ID, &share.FileID, &share.CreatedBy, &share.ShareType, &token,
		&passwordHash, &expiresAt, &maxDownloads, &share.DownloadCount,
		&share.AllowUpload, &share.PreviewOnly, &sharedWith, &share.Permission,
		&share.EncryptedKey, &share.IsActive, &share.ViewCount,
		&share.CreatedAt, &share.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("share not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get share: %w", err)
	}

	if token.Valid {
		share.Token = &token.String
	}
	if passwordHash.Valid {
		share.PasswordHash = &passwordHash.String
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		share.ExpiresAt = &t
	}
	if maxDownloads.Valid {
		md, _ := fmt.Sscanf(maxDownloads.String, "%d", &share.MaxDownloads)
		_ = md
	}
	if sharedWith.Valid {
		share.SharedWith = &sharedWith.String
	}

	return share, nil
}

// GetShareByToken retrieves a share by token.
func (m *Manager) GetShareByToken(ctx context.Context, token string) (*Share, error) {
	query := `SELECT id, file_id, created_by, share_type, token, password_hash,
		expires_at, max_downloads, download_count, allow_upload, preview_only,
		shared_with, permission, encrypted_key, is_active, view_count, created_at, updated_at
	FROM shares WHERE token = ? AND is_active = 1`

	share := &Share{}
	var passwordHash, expiresAt, maxDownloads, sharedWith sql.NullString

	err := m.db.QueryRowContext(ctx, query, token).Scan(
		&share.ID, &share.FileID, &share.CreatedBy, &share.ShareType, &share.Token,
		&passwordHash, &expiresAt, &maxDownloads, &share.DownloadCount,
		&share.AllowUpload, &share.PreviewOnly, &sharedWith, &share.Permission,
		&share.EncryptedKey, &share.IsActive, &share.ViewCount,
		&share.CreatedAt, &share.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("share not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get share: %w", err)
	}

	if passwordHash.Valid {
		share.PasswordHash = &passwordHash.String
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		share.ExpiresAt = &t
	}
	if maxDownloads.Valid {
		md, _ := fmt.Sscanf(maxDownloads.String, "%d", &share.MaxDownloads)
		_ = md
	}
	if sharedWith.Valid {
		share.SharedWith = &sharedWith.String
	}

	return share, nil
}

// GetSharesByFile retrieves all shares for a file.
func (m *Manager) GetSharesByFile(ctx context.Context, fileID string) ([]*Share, error) {
	query := `SELECT id, file_id, created_by, share_type, token, password_hash,
		expires_at, max_downloads, download_count, allow_upload, preview_only,
		shared_with, permission, encrypted_key, is_active, view_count, created_at, updated_at
	FROM shares WHERE file_id = ? AND is_active = 1 ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shares: %w", err)
	}
	defer rows.Close()

	shares := make([]*Share, 0)
	for rows.Next() {
		share := &Share{}
		var token, passwordHash, expiresAt, maxDownloads, sharedWith sql.NullString

		err := rows.Scan(
			&share.ID, &share.FileID, &share.CreatedBy, &share.ShareType, &token,
			&passwordHash, &expiresAt, &maxDownloads, &share.DownloadCount,
			&share.AllowUpload, &share.PreviewOnly, &sharedWith, &share.Permission,
			&share.EncryptedKey, &share.IsActive, &share.ViewCount,
			&share.CreatedAt, &share.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}

		if token.Valid {
			share.Token = &token.String
		}
		if passwordHash.Valid {
			share.PasswordHash = &passwordHash.String
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			share.ExpiresAt = &t
		}
		if maxDownloads.Valid {
			md, _ := fmt.Sscanf(maxDownloads.String, "%d", &share.MaxDownloads)
			_ = md
		}
		if sharedWith.Valid {
			share.SharedWith = &sharedWith.String
		}

		shares = append(shares, share)
	}

	return shares, nil
}

// GetSharesByUser retrieves all shares created by a user.
func (m *Manager) GetSharesByUser(ctx context.Context, userID string) ([]*Share, error) {
	query := `SELECT id, file_id, created_by, share_type, token, password_hash,
		expires_at, max_downloads, download_count, allow_upload, preview_only,
		shared_with, permission, encrypted_key, is_active, view_count, created_at, updated_at
	FROM shares WHERE created_by = ? ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shares: %w", err)
	}
	defer rows.Close()

	shares := make([]*Share, 0)
	for rows.Next() {
		share := &Share{}
		var token, passwordHash, expiresAt, maxDownloads, sharedWith sql.NullString

		err := rows.Scan(
			&share.ID, &share.FileID, &share.CreatedBy, &share.ShareType, &token,
			&passwordHash, &expiresAt, &maxDownloads, &share.DownloadCount,
			&share.AllowUpload, &share.PreviewOnly, &sharedWith, &share.Permission,
			&share.EncryptedKey, &share.IsActive, &share.ViewCount,
			&share.CreatedAt, &share.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}

		if token.Valid {
			share.Token = &token.String
		}
		if passwordHash.Valid {
			share.PasswordHash = &passwordHash.String
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			share.ExpiresAt = &t
		}
		if maxDownloads.Valid {
			md, _ := fmt.Sscanf(maxDownloads.String, "%d", &share.MaxDownloads)
			_ = md
		}
		if sharedWith.Valid {
			share.SharedWith = &sharedWith.String
		}

		shares = append(shares, share)
	}

	return shares, nil
}

// GetReceivedShares retrieves shares shared with a user.
func (m *Manager) GetReceivedShares(ctx context.Context, userID string) ([]*Share, error) {
	query := `SELECT id, file_id, created_by, share_type, token, password_hash,
		expires_at, max_downloads, download_count, allow_upload, preview_only,
		shared_with, permission, encrypted_key, is_active, view_count, created_at, updated_at
	FROM shares WHERE shared_with = ? AND is_active = 1 ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shares: %w", err)
	}
	defer rows.Close()

	shares := make([]*Share, 0)
	for rows.Next() {
		share := &Share{}
		var token, passwordHash, expiresAt, maxDownloads, sharedWith sql.NullString

		err := rows.Scan(
			&share.ID, &share.FileID, &share.CreatedBy, &share.ShareType, &token,
			&passwordHash, &expiresAt, &maxDownloads, &share.DownloadCount,
			&share.AllowUpload, &share.PreviewOnly, &sharedWith, &share.Permission,
			&share.EncryptedKey, &share.IsActive, &share.ViewCount,
			&share.CreatedAt, &share.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}

		if token.Valid {
			share.Token = &token.String
		}
		if passwordHash.Valid {
			share.PasswordHash = &passwordHash.String
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			share.ExpiresAt = &t
		}
		if maxDownloads.Valid {
			md, _ := fmt.Sscanf(maxDownloads.String, "%d", &share.MaxDownloads)
			_ = md
		}
		if sharedWith.Valid {
			share.SharedWith = &sharedWith.String
		}

		shares = append(shares, share)
	}

	return shares, nil
}

// UpdateShare updates a share.
func (m *Manager) UpdateShare(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	setClauses := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+2)

	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
		if b, ok := value.(bool); ok {
			args = append(args, boolToInt(b))
		} else {
			args = append(args, value)
		}
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE shares SET %s WHERE id = ?", setClauses)

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update share: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("share not found")
	}

	return nil
}

// RevokeShare deactivates a share.
func (m *Manager) RevokeShare(ctx context.Context, id string) error {
	return m.UpdateShare(ctx, id, map[string]any{
		"is_active": 0,
	})
}

// IncrementShareDownloadCount increments the download counter.
func (m *Manager) IncrementShareDownloadCount(ctx context.Context, id string) error {
	_, err := m.db.ExecContext(ctx,
		"UPDATE shares SET download_count = download_count + 1 WHERE id = ?",
		id,
	)
	return err
}

// IncrementShareViewCount increments the view counter.
func (m *Manager) IncrementShareViewCount(ctx context.Context, id string) error {
	_, err := m.db.ExecContext(ctx,
		"UPDATE shares SET view_count = view_count + 1 WHERE id = ?",
		id,
	)
	return err
}

// DeleteShare permanently deletes a share.
func (m *Manager) DeleteShare(ctx context.Context, id string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM shares WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete share: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("share not found")
	}

	return nil
}
