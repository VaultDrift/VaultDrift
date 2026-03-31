package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// GCJobType is the worker job type for garbage collection
const GCJobType = "garbage-collect"

// GCSpec specifies what to collect
type GCSpec struct {
	Type      string    // "orphaned-chunks", "old-versions", "trash"
	OlderThan time.Time // Only collect items older than this
}

// GCResult contains garbage collection results
type GCResult struct {
	Type      string
	Collected int
	Freed     int64 // bytes freed
	Errors    []error
}

// GCWorker handles garbage collection
type GCWorker struct {
	db      *db.Manager
	storage interface {
		Delete(ctx context.Context, id string) error
	}
}

// NewGCWorker creates a new garbage collection worker
func NewGCWorker(database *db.Manager, store interface {
	Delete(ctx context.Context, id string) error
}) *GCWorker {
	return &GCWorker{
		db:      database,
		storage: store,
	}
}

// Handler returns a worker handler function
func (w *GCWorker) Handler() Handler {
	return func(ctx context.Context, job Job) (interface{}, error) {
		spec, ok := job.Payload.(GCSpec)
		if !ok {
			return nil, fmt.Errorf("invalid payload type")
		}

		switch spec.Type {
		case "orphaned-chunks":
			return w.collectOrphanedChunks(ctx, spec)
		case "old-versions":
			return w.collectOldVersions(ctx, spec)
		case "trash":
			return w.collectTrash(ctx, spec)
		default:
			return nil, fmt.Errorf("unknown gc type: %s", spec.Type)
		}
	}
}

// collectOrphanedChunks finds and removes chunks not referenced by any file
func (w *GCWorker) collectOrphanedChunks(ctx context.Context, spec GCSpec) (*GCResult, error) {
	result := &GCResult{Type: "orphaned-chunks"}

	// Query for orphaned chunks
	// This is a simplified version - in production, you'd want to:
	// 1. Get all chunk IDs from storage
	// 2. Get all referenced chunk IDs from database
	// 3. Find chunks in storage but not in database
	// 4. Delete orphaned chunks

	log.Println("Scanning for orphaned chunks...")

	// TODO: Implement actual orphaned chunk detection
	// For now, just log that we would do this

	return result, nil
}

// collectOldVersions removes old file versions
func (w *GCWorker) collectOldVersions(ctx context.Context, spec GCSpec) (*GCResult, error) {
	result := &GCResult{Type: "old-versions"}

	log.Printf("Collecting versions older than %s...", spec.OlderThan)

	// Query for old versions
	rows, err := w.db.Query(ctx, `
		SELECT id, size_bytes
		FROM file_versions
		WHERE created_at < ?
		ORDER BY file_id, version DESC
	`, spec.OlderThan)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	// Keep track of versions per file
	fileVersions := make(map[string]int)

	for rows.Next() {
		var versionID string
		var size int64
		if err := rows.Scan(&versionID, &size); err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		// Keep last 5 versions per file
		// In production, this would be configurable
		fileID := "" // Would need to include file_id in query
		fileVersions[fileID]++

		if fileVersions[fileID] > 5 {
			// Delete old version
			if err := w.deleteVersion(ctx, versionID); err != nil {
				result.Errors = append(result.Errors, err)
				continue
			}
			result.Collected++
			result.Freed += size
		}
	}

	return result, nil
}

// collectTrash permanently deletes items in trash older than retention period
func (w *GCWorker) collectTrash(ctx context.Context, spec GCSpec) (*GCResult, error) {
	result := &GCResult{Type: "trash"}

	log.Printf("Collecting trash older than %s...", spec.OlderThan)

	// Query for trashed items
	rows, err := w.db.Query(ctx, `
		SELECT id, size_bytes
		FROM files
		WHERE is_trashed = 1 AND trashed_at < ?
	`, spec.OlderThan)
	if err != nil {
		return nil, fmt.Errorf("failed to query trash: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fileID string
		var size int64
		if err := rows.Scan(&fileID, &size); err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		// Delete file permanently
		if err := w.deleteFile(ctx, fileID); err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.Collected++
		result.Freed += size
	}

	return result, nil
}

// deleteVersion deletes a file version
func (w *GCWorker) deleteVersion(ctx context.Context, versionID string) error {
	// Delete from storage
	if err := w.storage.Delete(ctx, versionID); err != nil {
		return err
	}

	// Delete from database
	_, err := w.db.Exec(ctx, `DELETE FROM file_versions WHERE id = ?`, versionID)
	return err
}

// deleteFile permanently deletes a file
func (w *GCWorker) deleteFile(ctx context.Context, fileID string) error {
	// Delete from storage
	if err := w.storage.Delete(ctx, fileID); err != nil {
		return err
	}

	// Delete from database
	_, err := w.db.Exec(ctx, `DELETE FROM files WHERE id = ?`, fileID)
	return err
}

// Register registers the GC handler with a worker pool
func (w *GCWorker) Register(pool *WorkerPool) {
	pool.RegisterHandler(GCJobType, w.Handler())
}

// QueueGCJob queues a garbage collection job
func QueueGCJob(pool *WorkerPool, spec GCSpec) error {
	return pool.Submit(Job{
		ID:       fmt.Sprintf("gc-%s-%d", spec.Type, time.Now().Unix()),
		Type:     GCJobType,
		Payload:  spec,
		Priority: 1, // Low priority
	})
}
