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

	log.Println("Scanning for orphaned chunks...")

	// Get chunks with ref_count = 0
	orphanedChunks, err := w.db.ListOrphanedChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list orphaned chunks: %w", err)
	}

	log.Printf("Found %d orphaned chunks to collect", len(orphanedChunks))

	for _, chunk := range orphanedChunks {
		// Skip if chunk is newer than the retention period
		if spec.OlderThan.IsZero() || chunk.CreatedAt.After(spec.OlderThan) {
			continue
		}

		// Delete from storage first
		if err := w.storage.Delete(ctx, chunk.Hash); err != nil {
			log.Printf("Failed to delete chunk from storage: %s, error: %v", chunk.Hash, err)
			result.Errors = append(result.Errors, fmt.Errorf("failed to delete chunk %s from storage: %w", chunk.Hash, err))
			continue
		}

		// Delete from database
		if err := w.db.DeleteChunk(ctx, chunk.Hash); err != nil {
			log.Printf("Failed to delete chunk from database: %s, error: %v", chunk.Hash, err)
			result.Errors = append(result.Errors, fmt.Errorf("failed to delete chunk %s from database: %w", chunk.Hash, err))
			continue
		}

		result.Collected++
		result.Freed += chunk.SizeBytes
		log.Printf("Deleted orphaned chunk: %s (%d bytes)", chunk.Hash, chunk.SizeBytes)
	}

	log.Printf("Orphaned chunk collection complete: %d chunks freed, %d bytes", result.Collected, result.Freed)
	return result, nil
}

// collectOldVersions removes old file versions (manifests) keeping only the latest N per file.
func (w *GCWorker) collectOldVersions(ctx context.Context, spec GCSpec) (*GCResult, error) {
	result := &GCResult{Type: "old-versions"}

	log.Printf("Collecting versions older than %s...", spec.OlderThan)

	// Query manifests grouped by file, keeping track of count per file
	rows, err := w.db.Query(ctx, `
		SELECT id, file_id, size_bytes
		FROM manifests
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
		var manifestID string
		var fileID string
		var size int64
		if err := rows.Scan(&manifestID, &fileID, &size); err != nil {
			result.Errors = append(result.Errors, err)
            continue
        }

        // Keep last 5 versions per file
        fileVersions[fileID]++

        if fileVersions[fileID] > 5 {
            // Delete old manifest version
            if err := w.db.DeleteManifest(ctx, manifestID); err != nil {
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

        // Delete file permanently (cleans up chunks)
        if err := w.deleteFile(ctx, fileID); err != nil {
            result.Errors = append(result.Errors, err)
            continue
        }

        result.Collected++
        result.Freed += size
    }

    return result, nil
}

// deleteFile permanently deletes a file and decrements chunk ref counts
func (w *GCWorker) deleteFile(ctx context.Context, fileID string) error {
	// Get the latest manifest to find chunk hashes
	manifest, err := w.db.GetLatestManifest(ctx, fileID)
	if err == nil && manifest != nil {
		// Decrement ref count for each chunk and delete if orphaned
		for _, chunkHash := range manifest.Chunks {
		if err := w.db.DecrementRefCount(ctx, chunkHash); err != nil {
				log.Printf("GC: failed to decrement ref for chunk %s: %v", chunkHash, err)
                continue
            }
            // Check if chunk is now orphaned
            chunk, err := w.db.GetChunk(ctx, chunkHash)
            if err == nil && chunk.RefCount <= 0 {
                _ = w.storage.Delete(ctx, chunkHash)
                _ = w.db.DeleteChunk(ctx, chunkHash)
            }
        }
    }

    // Delete the file record from database
    _, _ = w.db.Exec(ctx, `DELETE FROM files WHERE id = ?`, fileID)
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
