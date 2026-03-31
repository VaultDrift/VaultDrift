package worker

import (
	"context"
	"log"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
)

// Manager manages background workers for the server
type Manager struct {
	pool      *WorkerPool
	scheduler *Scheduler
	db        *db.Manager
	storage   storage.Backend
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a new worker manager
func NewManager(database *db.Manager, store storage.Backend) *Manager {
	// Create job queue and worker pool
	queue := NewMemoryQueue()
	pool := NewWorkerPool(queue, 4) // 4 worker goroutines

	// Create GC worker
	gcWorker := NewGCWorker(database, store)
	gcWorker.Register(pool)

	// Create scheduler
	scheduler := NewScheduler(pool)

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		pool:      pool,
		scheduler: scheduler,
		db:        database,
		storage:   store,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the worker manager
func (m *Manager) Start() {
	log.Println("Starting background worker manager...")

	// Start worker pool
	m.pool.Start()

	// Start scheduler
	m.scheduler.Start()

	// Schedule recurring GC jobs
	m.scheduler.AddTask("trash-cleanup", GCJobType, GCSpec{
		Type:      "trash",
		OlderThan: time.Now().Add(-30 * 24 * time.Hour), // 30 days
	}, 24*time.Hour)

	// Start results processor
	go m.processResults()
}

// Stop stops the worker manager
func (m *Manager) Stop() {
	log.Println("Stopping background worker manager...")
	m.cancel()
	m.scheduler.Stop()
	m.pool.Stop()
}

// Pool returns the worker pool for registering additional handlers
func (m *Manager) Pool() *WorkerPool {
	return m.pool
}

// Scheduler returns the scheduler for adding tasks
func (m *Manager) Scheduler() *Scheduler {
	return m.scheduler
}

// QueueGC queues a garbage collection job
func (m *Manager) QueueGC(spec GCSpec) error {
	return QueueGCJob(m.pool, spec)
}

// processResults processes job results
func (m *Manager) processResults() {
	for result := range m.pool.Results() {
		if result.Error != nil {
			log.Printf("Job %s failed: %v", result.JobID, result.Error)
		} else {
			log.Printf("Job %s completed successfully", result.JobID)
		}
	}
}
