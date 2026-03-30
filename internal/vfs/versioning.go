package vfs

import (
	"context"
	"fmt"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// VersioningService handles file versioning operations.
type VersioningService struct {
	vfs *VFS
	db  *db.Manager
}

// FileVersion represents a specific version of a file.
type FileVersion struct {
	ID        string    `json:"id"`
	FileID    string    `json:"file_id"`
	Version   int       `json:"version"`
	SizeBytes int64     `json:"size_bytes"`
	ChunkHash string    `json:"chunk_hash"` // Hash of all chunks
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	Comment   string    `json:"comment,omitempty"`
}

// NewVersioningService creates a new versioning service.
func NewVersioningService(vfs *VFS, db *db.Manager) *VersioningService {
	return &VersioningService{vfs: vfs, db: db}
}

// CreateVersion creates a new version record for a file.
// This is called after a file is modified/uploaded.
func (vs *VersioningService) CreateVersion(ctx context.Context, fileID, userID string, sizeBytes int64, comment string) (*FileVersion, error) {
	// Get current file
	file, err := vs.db.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Increment version
	newVersion := file.Version + 1

	// Update file's current version
	if err := vs.db.UpdateFile(ctx, fileID, map[string]any{
		"version":    newVersion,
		"updated_at": time.Now().UTC(),
	}); err != nil {
		return nil, fmt.Errorf("failed to update file version: %w", err)
	}

	version := &FileVersion{
		ID:        generateVersionID(fileID, newVersion),
		FileID:    fileID,
		Version:   newVersion,
		SizeBytes: sizeBytes,
		CreatedAt: time.Now().UTC(),
		CreatedBy: userID,
		Comment:   comment,
	}

	return version, nil
}

// GetCurrentVersion returns the current version number of a file.
func (vs *VersioningService) GetCurrentVersion(ctx context.Context, fileID string) (int, error) {
	file, err := vs.db.GetFileByID(ctx, fileID)
	if err != nil {
		return 0, ErrNotFound
	}
	return file.Version, nil
}

// IncrementVersion increments the version number of a file.
func (vs *VersioningService) IncrementVersion(ctx context.Context, fileID string) (int, error) {
	file, err := vs.db.GetFileByID(ctx, fileID)
	if err != nil {
		return 0, ErrNotFound
	}

	newVersion := file.Version + 1
	if err := vs.db.UpdateFile(ctx, fileID, map[string]any{
		"version": newVersion,
	}); err != nil {
		return 0, err
	}

	return newVersion, nil
}

// ResetVersion resets a file's version to 1 (e.g., after restore).
func (vs *VersioningService) ResetVersion(ctx context.Context, fileID string) error {
	return vs.db.UpdateFile(ctx, fileID, map[string]any{
		"version": 1,
	})
}

func generateVersionID(fileID string, version int) string {
	return fmt.Sprintf("%s_v%d", fileID, version)
}
