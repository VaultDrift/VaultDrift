package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateChunk creates a new chunk record.
func (m *Manager) CreateChunk(ctx context.Context, chunk *Chunk) error {
	query := `INSERT INTO chunks (hash, size_bytes, storage_backend, storage_path, ref_count, is_encrypted, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		chunk.Hash, chunk.SizeBytes, chunk.StorageBackend, chunk.StoragePath,
		chunk.RefCount, boolToInt(chunk.IsEncrypted), chunk.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create chunk: %w", err)
	}

	return nil
}

// GetChunk retrieves a chunk by hash.
func (m *Manager) GetChunk(ctx context.Context, hash string) (*Chunk, error) {
	query := `SELECT hash, size_bytes, storage_backend, storage_path, ref_count, is_encrypted, created_at
	FROM chunks WHERE hash = ?`

	chunk := &Chunk{}
	var createdAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, hash).Scan(
		&chunk.Hash, &chunk.SizeBytes, &chunk.StorageBackend, &chunk.StoragePath,
		&chunk.RefCount, &chunk.IsEncrypted, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chunk not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if createdAt.Valid {
		chunk.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}

	return chunk, nil
}

// ChunkExists checks if a chunk exists.
func (m *Manager) ChunkExists(ctx context.Context, hash string) (bool, error) {
	var count int
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks WHERE hash = ?", hash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check chunk existence: %w", err)
	}
	return count > 0, nil
}

// IncrementRefCount increments the reference count of a chunk.
func (m *Manager) IncrementRefCount(ctx context.Context, hash string) error {
	result, err := m.db.ExecContext(ctx,
		"UPDATE chunks SET ref_count = ref_count + 1 WHERE hash = ?",
		hash,
	)
	if err != nil {
		return fmt.Errorf("failed to increment ref count: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("chunk not found")
	}

	return nil
}

// DecrementRefCount decrements the reference count of a chunk.
func (m *Manager) DecrementRefCount(ctx context.Context, hash string) error {
	result, err := m.db.ExecContext(ctx,
		"UPDATE chunks SET ref_count = ref_count - 1 WHERE hash = ?",
		hash,
	)
	if err != nil {
		return fmt.Errorf("failed to decrement ref count: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("chunk not found")
	}

	return nil
}

// ListOrphanedChunks retrieves chunks with ref_count = 0.
func (m *Manager) ListOrphanedChunks(ctx context.Context) ([]*Chunk, error) {
	query := `SELECT hash, size_bytes, storage_backend, storage_path, ref_count, is_encrypted, created_at
	FROM chunks WHERE ref_count = 0
	ORDER BY created_at ASC`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list orphaned chunks: %w", err)
	}
	defer rows.Close()

	chunks := make([]*Chunk, 0)
	for rows.Next() {
		chunk := &Chunk{}
		var createdAt sql.NullString

		err := rows.Scan(
			&chunk.Hash, &chunk.SizeBytes, &chunk.StorageBackend, &chunk.StoragePath,
			&chunk.RefCount, &chunk.IsEncrypted, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		if createdAt.Valid {
			chunk.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// DeleteChunk permanently deletes a chunk record.
func (m *Manager) DeleteChunk(ctx context.Context, hash string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM chunks WHERE hash = ?", hash)
	if err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("chunk not found")
	}

	return nil
}

// GetTotalChunkCount returns the total number of chunks.
func (m *Manager) GetTotalChunkCount(ctx context.Context) (int64, error) {
	var count int64
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get chunk count: %w", err)
	}
	return count, nil
}

// GetTotalChunkSize returns the total size of all chunks.
func (m *Manager) GetTotalChunkSize(ctx context.Context) (int64, error) {
	var size int64
	err := m.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size_bytes), 0) FROM chunks").Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("failed to get total chunk size: %w", err)
	}
	return size, nil
}
