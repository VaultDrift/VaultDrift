package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CreateFile creates a new file or folder.
func (m *Manager) CreateFile(ctx context.Context, file *File) error {
	query := `INSERT INTO files (
		id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		file.ID, file.UserID, file.ParentID, file.Name, file.NameEncrypted,
		file.Type, file.SizeBytes, file.MimeType, file.ManifestID, file.Checksum,
		boolToInt(file.IsEncrypted), file.EncryptedKey, boolToInt(file.IsTrashed), nil,
		file.Version, file.CreatedAt.Format(time.RFC3339), file.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("file already exists")
		}
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

// GetFileByID retrieves a file by ID.
func (m *Manager) GetFileByID(ctx context.Context, id string) (*File, error) {
	query := `SELECT id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	FROM files WHERE id = ?`

	file := &File{}
	var parentID, manifestID, checksum, trashedAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&file.ID, &file.UserID, &parentID, &file.Name, &file.NameEncrypted,
		&file.Type, &file.SizeBytes, &file.MimeType, &manifestID, &checksum,
		&file.IsEncrypted, &file.EncryptedKey, &file.IsTrashed, &trashedAt,
		&file.Version, &file.CreatedAt, &file.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if parentID.Valid {
		file.ParentID = &parentID.String
	}
	if manifestID.Valid {
		file.ManifestID = &manifestID.String
	}
	if checksum.Valid {
		file.Checksum = &checksum.String
	}
	if trashedAt.Valid {
		t, _ := time.Parse(time.RFC3339, trashedAt.String)
		file.TrashedAt = &t
	}

	return file, nil
}

// GetFileByPath retrieves a file by user ID, parent ID, and name.
func (m *Manager) GetFileByPath(ctx context.Context, userID string, parentID *string, name string) (*File, error) {
	query := `SELECT id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	FROM files WHERE user_id = ? AND parent_id IS ? AND name = ? AND is_trashed = 0`

	file := &File{}
	var parentIDVal, manifestID, checksum, trashedAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, userID, parentID, name).Scan(
		&file.ID, &file.UserID, &parentIDVal, &file.Name, &file.NameEncrypted,
		&file.Type, &file.SizeBytes, &file.MimeType, &manifestID, &checksum,
		&file.IsEncrypted, &file.EncryptedKey, &file.IsTrashed, &trashedAt,
		&file.Version, &file.CreatedAt, &file.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if parentIDVal.Valid {
		file.ParentID = &parentIDVal.String
	}
	if manifestID.Valid {
		file.ManifestID = &manifestID.String
	}
	if checksum.Valid {
		file.Checksum = &checksum.String
	}
	if trashedAt.Valid {
		t, _ := time.Parse(time.RFC3339, trashedAt.String)
		file.TrashedAt = &t
	}

	return file, nil
}

// ListDirectory retrieves files in a directory.
func (m *Manager) ListDirectory(ctx context.Context, userID string, parentID *string, opts ListOpts) ([]*File, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	query := `SELECT id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	FROM files
	WHERE user_id = ? AND parent_id IS ? AND is_trashed = 0
	ORDER BY type DESC, name ASC
	LIMIT ? OFFSET ?`

	rows, err := m.db.QueryContext(ctx, query, userID, parentID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}
	defer rows.Close()

	files := make([]*File, 0)
	for rows.Next() {
		file := &File{}
		var parentIDVal, manifestID, checksum, trashedAt sql.NullString

		err := rows.Scan(
			&file.ID, &file.UserID, &parentIDVal, &file.Name, &file.NameEncrypted,
			&file.Type, &file.SizeBytes, &file.MimeType, &manifestID, &checksum,
			&file.IsEncrypted, &file.EncryptedKey, &file.IsTrashed, &trashedAt,
			&file.Version, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if parentIDVal.Valid {
			file.ParentID = &parentIDVal.String
		}
		if manifestID.Valid {
			file.ManifestID = &manifestID.String
		}
		if checksum.Valid {
			file.Checksum = &checksum.String
		}
		if trashedAt.Valid {
			t, _ := time.Parse(time.RFC3339, trashedAt.String)
			file.TrashedAt = &t
		}

		files = append(files, file)
	}

	return files, nil
}

// UpdateFile updates a file's fields.
func (m *Manager) UpdateFile(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	allowedFields := map[string]bool{
		"name":           true,
		"name_encrypted": true,
		"size_bytes":     true,
		"mime_type":      true,
		"manifest_id":    true,
		"checksum":       true,
		"is_encrypted":   true,
		"encrypted_key":  true,
		"version":        true,
	}

	setClauses := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+2)

	for field, value := range updates {
		if !allowedFields[field] {
			return fmt.Errorf("invalid update field: %s", field)
		}
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

	query := fmt.Sprintf("UPDATE files SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// MoveFile moves a file to a new parent and/or renames it.
func (m *Manager) MoveFile(ctx context.Context, id, newParentID, newName string) error {
	query := `UPDATE files SET parent_id = ?, name = ?, updated_at = ? WHERE id = ?`
	result, err := m.db.ExecContext(ctx, query, newParentID, newName, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("file already exists at destination")
		}
		return fmt.Errorf("failed to move file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// SoftDelete marks a file as trashed.
func (m *Manager) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE files SET is_trashed = 1, trashed_at = ?, updated_at = ? WHERE id = ?`
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.ExecContext(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// ListTrash retrieves trashed files for a user.
func (m *Manager) ListTrash(ctx context.Context, userID string) ([]*File, error) {
	query := `SELECT id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	FROM files
	WHERE user_id = ? AND is_trashed = 1
	ORDER BY trashed_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list trash: %w", err)
	}
	defer rows.Close()

	files := make([]*File, 0)
	for rows.Next() {
		file := &File{}
		var parentID, manifestID, checksum, trashedAt sql.NullString

		err := rows.Scan(
			&file.ID, &file.UserID, &parentID, &file.Name, &file.NameEncrypted,
			&file.Type, &file.SizeBytes, &file.MimeType, &manifestID, &checksum,
			&file.IsEncrypted, &file.EncryptedKey, &file.IsTrashed, &trashedAt,
			&file.Version, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if parentID.Valid {
			file.ParentID = &parentID.String
		}
		if manifestID.Valid {
			file.ManifestID = &manifestID.String
		}
		if checksum.Valid {
			file.Checksum = &checksum.String
		}
		if trashedAt.Valid {
			t, _ := time.Parse(time.RFC3339, trashedAt.String)
			file.TrashedAt = &t
		}

		files = append(files, file)
	}

	return files, nil
}

// RestoreFromTrash restores a file from trash.
func (m *Manager) RestoreFromTrash(ctx context.Context, id string) error {
	query := `UPDATE files SET is_trashed = 0, trashed_at = NULL, updated_at = ? WHERE id = ?`
	result, err := m.db.ExecContext(ctx, query, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// PermanentDelete permanently deletes a file.
func (m *Manager) PermanentDelete(ctx context.Context, id string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM files WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// SearchFiles searches files by name for a user.
func (m *Manager) SearchFiles(ctx context.Context, userID, query string, limit int) ([]*File, error) {
	if limit <= 0 {
		limit = 100
	}

	sqlQuery := `SELECT id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	FROM files
	WHERE user_id = ? AND is_trashed = 0 AND name LIKE ?
	ORDER BY name ASC
	LIMIT ?`

	rows, err := m.db.QueryContext(ctx, sqlQuery, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}
	defer rows.Close()

	files := make([]*File, 0)
	for rows.Next() {
		file := &File{}
		var parentID, manifestID, checksum, trashedAt sql.NullString

		err := rows.Scan(
			&file.ID, &file.UserID, &parentID, &file.Name, &file.NameEncrypted,
			&file.Type, &file.SizeBytes, &file.MimeType, &manifestID, &checksum,
			&file.IsEncrypted, &file.EncryptedKey, &file.IsTrashed, &trashedAt,
			&file.Version, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if parentID.Valid {
			file.ParentID = &parentID.String
		}
		if manifestID.Valid {
			file.ManifestID = &manifestID.String
		}
		if checksum.Valid {
			file.Checksum = &checksum.String
		}
		if trashedAt.Valid {
			t, _ := time.Parse(time.RFC3339, trashedAt.String)
			file.TrashedAt = &t
		}

		files = append(files, file)
	}

	return files, nil
}

// RecentFiles retrieves recently modified files for a user.
func (m *Manager) RecentFiles(ctx context.Context, userID string, limit int) ([]*File, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, user_id, parent_id, name, name_encrypted, type,
		size_bytes, mime_type, manifest_id, checksum,
		is_encrypted, encrypted_key, is_trashed, trashed_at,
		version, created_at, updated_at
	FROM files
	WHERE user_id = ? AND is_trashed = 0 AND type = 'file'
	ORDER BY updated_at DESC
	LIMIT ?`

	rows, err := m.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent files: %w", err)
	}
	defer rows.Close()

	files := make([]*File, 0)
	for rows.Next() {
		file := &File{}
		var parentID, manifestID, checksum, trashedAt sql.NullString

		err := rows.Scan(
			&file.ID, &file.UserID, &parentID, &file.Name, &file.NameEncrypted,
			&file.Type, &file.SizeBytes, &file.MimeType, &manifestID, &checksum,
			&file.IsEncrypted, &file.EncryptedKey, &file.IsTrashed, &trashedAt,
			&file.Version, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if parentID.Valid {
			file.ParentID = &parentID.String
		}
		if manifestID.Valid {
			file.ManifestID = &manifestID.String
		}
		if checksum.Valid {
			file.Checksum = &checksum.String
		}
		if trashedAt.Valid {
			t, _ := time.Parse(time.RFC3339, trashedAt.String)
			file.TrashedAt = &t
		}

		files = append(files, file)
	}

	return files, nil
}
