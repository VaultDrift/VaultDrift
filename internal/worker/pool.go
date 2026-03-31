package worker

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// Job represents a unit of work
type Job struct {
	ID       string
	Type     string
	Payload  interface{}
	Priority int // Higher = more important
	Retries  int
	MaxRetries int
}

// Result represents a job execution result
type Result struct {
	JobID   string
	Success bool
	Error   error
	Output  interface{}
}

// Handler is a function that processes a job
type Handler func(ctx context.Context, job Job) (interface{}, error)

// Queue is a job queue interface
type Queue interface {
	Push(job Job) error
	Pop() (Job, bool)
	Len() int
}

// MemoryQueue is an in-memory job queue
type MemoryQueue struct {
	mu    sync.Mutex
	jobs  []Job
	cond  *sync.Cond
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	q := &MemoryQueue{
		jobs: make([]Job, 0),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Push adds a job to the queue
func (q *MemoryQueue) Push(job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Insert by priority (higher priority first)
	inserted := false
	for i, j := range q.jobs {
		if job.Priority > j.Priority {
			q.jobs = append(q.jobs[:i], append([]Job{job}, q.jobs[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.jobs = append(q.jobs, job)
	}

	q.cond.Signal()
	return nil
}

// Pop removes and returns the highest priority job
func (q *MemoryQueue) Pop() (Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.jobs) == 0 {
		q.cond.Wait()
	}

	if len(q.jobs) == 0 {
		return Job{}, false
	}

	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, true
}

// Len returns the queue length
func (q *MemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	queue      Queue
	handlers   map[string]Handler
	workers    int
	results    chan Result
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue Queue, workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		queue:    queue,
		handlers: make(map[string]Handler),
		workers:  workers,
		results:  make(chan Result, 100),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHandler registers a handler for a job type
func (p *WorkerPool) RegisterHandler(jobType string, handler Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[jobType] = handler
}

// Submit submits a job to the queue
func (p *WorkerPool) Submit(job Job) error {
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}
	return p.queue.Push(job)
}

// Results returns the results channel
func (p *WorkerPool) Results() <-chan Result {
	return p.results
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	log.Printf("Starting worker pool with %d workers", p.workers)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	p.cancel()
	p.wg.Wait()
	close(p.results)

	log.Println("Worker pool stopped")
}

// worker is the main worker loop
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	log.Printf("Worker %d started", id)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		default:
		}

		job, ok := p.queue.Pop()
		if !ok {
			continue
		}

		p.processJob(job)
	}
}

// processJob processes a single job
func (p *WorkerPool) processJob(job Job) {
	p.mu.RLock()
	handler, exists := p.handlers[job.Type]
	p.mu.RUnlock()

	if !exists {
		p.results <- Result{
			JobID:   job.ID,
			Success: false,
			Error:   fmt.Errorf("no handler for job type: %s", job.Type),
		}
		return
	}

	// Create timeout context for job
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	output, err := handler(ctx, job)

	if err != nil && job.Retries < job.MaxRetries {
		// Retry the job
		job.Retries++
		log.Printf("Job %s failed, retrying (%d/%d): %v", job.ID, job.Retries, job.MaxRetries, err)
		time.Sleep(time.Second * time.Duration(job.Retries))
		p.queue.Push(job)
		return
	}

	p.results <- Result{
		JobID:   job.ID,
		Success: err == nil,
		Error:   err,
		Output:  output,
	}
}

// Scheduler schedules recurring jobs
type Scheduler struct {
	pool   *WorkerPool
	tasks  map[string]*scheduledTask
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type scheduledTask struct {
	jobType  string
	payload  interface{}
	interval time.Duration
	lastRun  time.Time
}

// NewScheduler creates a new scheduler
func NewScheduler(pool *WorkerPool) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		pool:  pool,
		tasks: make(map[string]*scheduledTask),
		ctx:   ctx,
		cancel: cancel,
	}
}

// AddTask adds a recurring task
func (s *Scheduler) AddTask(name, jobType string, payload interface{}, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[name] = &scheduledTask{
		jobType:  jobType,
		payload:  payload,
		interval: interval,
	}
}

// RemoveTask removes a scheduled task
func (s *Scheduler) RemoveTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, name)
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// run is the scheduler main loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndRunTasks()
		}
	}
}

// checkAndRunTasks checks for tasks that need to run
func (s *Scheduler) checkAndRunTasks() {
	s.mu.RLock()
	tasks := make(map[string]*scheduledTask)
	for k, v := range s.tasks {
		tasks[k] = v
	}
	s.mu.RUnlock()

	now := time.Now()
	for name, task := range tasks {
		if now.Sub(task.lastRun) >= task.interval {
			s.pool.Submit(Job{
				ID:       fmt.Sprintf("scheduled-%s-%d", name, now.Unix()),
				Type:     task.jobType,
				Payload:  task.payload,
				Priority: 0, // Low priority for scheduled tasks
			})

			s.mu.Lock()
			if t, exists := s.tasks[name]; exists {
				t.lastRun = now
			}
			s.mu.Unlock()
		}
	}
}
