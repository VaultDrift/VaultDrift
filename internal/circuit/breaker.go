// Package circuit provides circuit breaker pattern implementation for resilient
// external service calls.
package circuit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed means the circuit is closed and requests flow normally.
	StateClosed State = iota
	// StateOpen means the circuit is open and requests fail fast.
	StateOpen
	// StateHalfOpen means the circuit is testing if the service recovered.
	StateHalfOpen
)

// String returns the state name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration.
type Config struct {
	// MaxFailures before opening the circuit.
	MaxFailures uint32
	// Timeout after which the circuit moves to half-open.
	Timeout time.Duration
	// MaxRetries for half-open test requests.
	MaxRetries int
	// RetryDelay between retries.
	RetryDelay time.Duration
}

// DefaultConfig returns a reasonable default configuration.
func DefaultConfig() Config {
	return Config{
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  1 * time.Second,
	}
}

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	name   string
	config Config

	mu           sync.RWMutex
	state        State
	failures     uint32
	lastFailure  time.Time
	halfOpenCalls uint32
}

// ErrOpenCircuit is returned when the circuit is open.
var ErrOpenCircuit = errors.New("circuit breaker is open")

// ErrTimeout is returned when the call times out.
var ErrTimeout = errors.New("circuit breaker timeout")

// New creates a new circuit breaker.
func New(name string, config Config) *Breaker {
	return &Breaker{
		name:   name,
		config: config,
		state:  StateClosed,
	}
}

// Name returns the circuit breaker name.
func (cb *Breaker) Name() string {
	return cb.name
}

// State returns the current state.
func (cb *Breaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Execute runs the given function with circuit breaker protection.
func (cb *Breaker) Execute(ctx context.Context, fn func() error) error {
	state := cb.currentState()

	if state == StateOpen {
		return fmt.Errorf("%s: %w", cb.name, ErrOpenCircuit)
	}

	// For half-open state, limit concurrent test requests
	if state == StateHalfOpen {
		if !cb.acquireHalfOpenSlot() {
			return fmt.Errorf("%s: %w", cb.name, ErrOpenCircuit)
		}
		defer cb.releaseHalfOpenSlot()
	}

	// Execute with retries
	err := cb.executeWithRetries(ctx, fn)

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// ExecuteWithResult runs the given function that returns a result.
func ExecuteWithResult[T any](cb *Breaker, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	state := cb.currentState()

	if state == StateOpen {
		return result, fmt.Errorf("%s: %w", cb.name, ErrOpenCircuit)
	}

	if state == StateHalfOpen {
		if !cb.acquireHalfOpenSlot() {
			return result, fmt.Errorf("%s: %w", cb.name, ErrOpenCircuit)
		}
		defer cb.releaseHalfOpenSlot()
	}

	err := cb.executeWithRetries(ctx, func() error {
		var err error
		result, err = fn()
		return err
	})

	if err != nil {
		cb.recordFailure()
		return result, err
	}

	cb.recordSuccess()
	return result, nil
}

func (cb *Breaker) currentState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Since(cb.lastFailure) > cb.config.Timeout {
		cb.state = StateHalfOpen
		cb.halfOpenCalls = 0
	}

	return cb.state
}

func (cb *Breaker) acquireHalfOpenSlot() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.halfOpenCalls < 1 { // Allow only 1 test request at a time
		cb.halfOpenCalls++
		return true
	}
	return false
}

func (cb *Breaker) releaseHalfOpenSlot() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.halfOpenCalls--
}

func (cb *Breaker) executeWithRetries(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cb.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(cb.config.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		done := make(chan error, 1)
		go func() {
			done <- fn()
		}()

		select {
		case err := <-done:
			if err == nil {
				return nil
			}
			lastErr = err
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cb.config.Timeout):
			lastErr = ErrTimeout
		}
	}

	return fmt.Errorf("after %d retries: %w", cb.config.MaxRetries, lastErr)
}

func (cb *Breaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.halfOpenCalls = 0
	}
}

func (cb *Breaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.halfOpenCalls = 0
	} else if cb.failures >= cb.config.MaxFailures {
		cb.state = StateOpen
	}
}

// Stats returns circuit breaker statistics.
func (cb *Breaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":          cb.name,
		"state":         cb.state.String(),
		"failures":      cb.failures,
		"last_failure":  cb.lastFailure,
		"half_open_calls": cb.halfOpenCalls,
	}
}

// Registry manages multiple circuit breakers.
type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*Breaker
}

// NewRegistry creates a new circuit breaker registry.
func NewRegistry() *Registry {
	return &Registry{
		breakers: make(map[string]*Breaker),
	}
}

// GetOrCreate returns an existing breaker or creates a new one.
func (r *Registry) GetOrCreate(name string, config Config) *Breaker {
	r.mu.RLock()
	if cb, ok := r.breakers[name]; ok {
		r.mu.RUnlock()
		return cb
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := r.breakers[name]; ok {
		return cb
	}

	cb := New(name, config)
	r.breakers[name] = cb
	return cb
}

// Get returns an existing breaker or nil.
func (r *Registry) Get(name string) (*Breaker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cb, ok := r.breakers[name]
	return cb, ok
}

// AllStats returns stats for all circuit breakers.
func (r *Registry) AllStats() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make([]map[string]interface{}, 0, len(r.breakers))
	for _, cb := range r.breakers {
		stats = append(stats, cb.Stats())
	}
	return stats
}
