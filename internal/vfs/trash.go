package vfs

import (
	"context"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// TrashService handles trash operations.
type TrashService struct {
	vfs *VFS
	db  *db.Manager
}

// NewTrashService creates a new trash service.
func NewTrashService(vfs *VFS, db *db.Manager) *TrashService {
	return &TrashService{vfs: vfs, db: db}
}

// List returns all items in trash for a user.
func (t *TrashService) List(ctx context.Context, userID string) ([]*db.File, error) {
	return t.vfs.ListTrash(ctx, userID)
}

// Restore restores an item from trash.
func (t *TrashService) Restore(ctx context.Context, fileID string) error {
	return t.vfs.Restore(ctx, fileID)
}

// Empty permanently deletes all items in trash for a user.
func (t *TrashService) Empty(ctx context.Context, userID string) error {
	items, err := t.vfs.ListTrash(ctx, userID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := t.vfs.DeletePermanent(ctx, item.ID); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// CleanupExpired removes trash items older than retention period.
func (t *TrashService) CleanupExpired(ctx context.Context, retentionDays int) error {
	// Get all trashed items older than retention period
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	query := `SELECT id FROM files WHERE is_trashed = 1 AND trashed_at < ?`
	rows, err := t.db.Query(ctx, query, cutoff.Format(time.RFC3339))
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		if err := t.vfs.DeletePermanent(ctx, id); err != nil {
			continue
		}
	}

	return nil
}
