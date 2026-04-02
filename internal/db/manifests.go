package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CreateManifest creates a new manifest.
func (m *Manager) CreateManifest(ctx context.Context, manifest *Manifest) error {
	chunksJSON, err := json.Marshal(manifest.Chunks)
	if err != nil {
		return fmt.Errorf("failed to marshal chunks: %w", err)
	}

	query := `INSERT INTO manifests (id, file_id, version, size_bytes, chunk_count, chunks, checksum, device_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = m.db.ExecContext(ctx, query,
		manifest.ID, manifest.FileID, manifest.Version, manifest.SizeBytes,
		manifest.ChunkCount, string(chunksJSON), manifest.Checksum, manifest.DeviceID,
		manifest.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}

	return nil
}

// GetManifest retrieves a manifest by ID.
func (m *Manager) GetManifest(ctx context.Context, id string) (*Manifest, error) {
	query := `SELECT id, file_id, version, size_bytes, chunk_count, chunks, checksum, device_id, created_at
	FROM manifests WHERE id = ?`

	manifest := &Manifest{}
	var chunksJSON string
	var createdAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&manifest.ID, &manifest.FileID, &manifest.Version, &manifest.SizeBytes,
		&manifest.ChunkCount, &chunksJSON, &manifest.Checksum, &manifest.DeviceID, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("manifest not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if createdAt.Valid {
		manifest.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}

	if err := json.Unmarshal([]byte(chunksJSON), &manifest.Chunks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunks: %w", err)
	}

	return manifest, nil
}

// GetLatestManifest retrieves the latest manifest for a file.
func (m *Manager) GetLatestManifest(ctx context.Context, fileID string) (*Manifest, error) {
	query := `SELECT id, file_id, version, size_bytes, chunk_count, chunks, checksum, device_id, created_at
	FROM manifests WHERE file_id = ? ORDER BY version DESC LIMIT 1`

	manifest := &Manifest{}
	var chunksJSON string
	var createdAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, fileID).Scan(
		&manifest.ID, &manifest.FileID, &manifest.Version, &manifest.SizeBytes,
		&manifest.ChunkCount, &chunksJSON, &manifest.Checksum, &manifest.DeviceID, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("manifest not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if createdAt.Valid {
		manifest.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}

	if err := json.Unmarshal([]byte(chunksJSON), &manifest.Chunks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunks: %w", err)
	}

	return manifest, nil
}

// ListVersions retrieves all manifests for a file.
func (m *Manager) ListVersions(ctx context.Context, fileID string) ([]*Manifest, error) {
	query := `SELECT id, file_id, version, size_bytes, chunk_count, chunks, checksum, device_id, created_at
	FROM manifests WHERE file_id = ? ORDER BY version DESC`

	rows, err := m.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	manifests := make([]*Manifest, 0)
	for rows.Next() {
		manifest := &Manifest{}
		var chunksJSON string
		var createdAt sql.NullString

		err := rows.Scan(
			&manifest.ID, &manifest.FileID, &manifest.Version, &manifest.SizeBytes,
			&manifest.ChunkCount, &chunksJSON, &manifest.Checksum, &manifest.DeviceID, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan manifest: %w", err)
		}

		if createdAt.Valid {
			manifest.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}

		if err := json.Unmarshal([]byte(chunksJSON), &manifest.Chunks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chunks: %w", err)
		}

		manifests = append(manifests, manifest)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate manifests: %w", err)
	}

	return manifests, nil
}

// DeleteManifest permanently deletes a manifest.
func (m *Manager) DeleteManifest(ctx context.Context, id string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM manifests WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("manifest not found")
	}

	return nil
}
