package thumbnail

import (
	"context"
	"fmt"

	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/worker"
)

// JobType is the worker job type for thumbnail generation
const JobType = "thumbnail"

// JobPayload is the payload for thumbnail jobs
type JobPayload struct {
	FileID   string
	MimeType string
}

// Worker handles thumbnail generation jobs
type Worker struct {
	generator *Generator
	storage   storage.Backend
}

// NewWorker creates a new thumbnail worker
func NewWorker(generator *Generator, store storage.Backend) *Worker {
	return &Worker{
		generator: generator,
		storage:   store,
	}
}

// Handler returns a worker handler function
func (w *Worker) Handler() worker.Handler {
	return func(ctx context.Context, job worker.Job) (interface{}, error) {
		payload, ok := job.Payload.(JobPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type")
		}

		// Check if thumbnail already exists
		if _, exists := w.generator.Get(payload.FileID, SizeMedium.Name); exists {
			return nil, nil // Already generated
		}

		// Get file data from storage
		data, err := w.storage.Get(ctx, payload.FileID)
		if err != nil {
			return nil, fmt.Errorf("failed to get file: %w", err)
		}

		// Generate thumbnails
		paths, err := w.generator.Generate(payload.FileID, payload.MimeType, data)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thumbnails: %w", err)
		}

		return paths, nil
	}
}

// Register registers the thumbnail handler with a worker pool
func (w *Worker) Register(pool *worker.WorkerPool) {
	pool.RegisterHandler(JobType, w.Handler())
}

// QueueJob queues a thumbnail generation job
func QueueJob(pool *worker.WorkerPool, fileID, mimeType string) error {
	return pool.Submit(worker.Job{
		ID:       fmt.Sprintf("thumb-%s", fileID),
		Type:     JobType,
		Payload:  JobPayload{FileID: fileID, MimeType: mimeType},
		Priority: 5, // Medium priority
	})
}
