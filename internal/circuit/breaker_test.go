package circuit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	config := Config{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
		MaxRetries:  0,
		RetryDelay:  0,
	}

	cb := New("test", config)

	// Initial state should be closed
	if cb.State() != StateClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.State())
	}

	// Failures up to MaxFailures should keep circuit closed
	for i := 0; i < int(config.MaxFailures)-1; i++ {
		err := cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
		if err == nil {
			t.Error("Expected error")
		}
		if cb.State() != StateClosed {
			t.Errorf("Expected state Closed after %d failures, got %v", i+1, cb.State())
		}
	}

	// One more failure should open the circuit
	err := cb.Execute(context.Background(), func() error {
		return errors.New("failure")
	})
	if err == nil {
		t.Error("Expected error")
	}
	if cb.State() != StateOpen {
		t.Errorf("Expected state Open after max failures, got %v", cb.State())
	}

	// Requests should fail fast when open
	err = cb.Execute(context.Background(), func() error {
		return nil
	})
	if !errors.Is(err, ErrOpenCircuit) {
		t.Errorf("Expected ErrOpenCircuit, got %v", err)
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	config := Config{
		MaxFailures: 2,
		Timeout:     50 * time.Millisecond,
		MaxRetries:  0,
		RetryDelay:  0,
	}

	cb := New("test", config)

	// Open the circuit
	for i := 0; i < int(config.MaxFailures); i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("Expected state Open, got %v", cb.State())
	}

	// Wait for timeout to transition to half-open
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// State should transition to half-open on next check
	_ = cb.currentState()
	if cb.State() != StateHalfOpen {
		t.Errorf("Expected state HalfOpen after timeout, got %v", cb.State())
	}

	// Success should close the circuit
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed after success, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	config := Config{
		MaxFailures: 2,
		Timeout:     50 * time.Millisecond,
		MaxRetries:  0,
		RetryDelay:  0,
	}

	cb := New("test", config)

	// Open the circuit
	for i := 0; i < int(config.MaxFailures); i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Failure in half-open should reopen
	err := cb.Execute(context.Background(), func() error {
		return errors.New("failure")
	})
	if err == nil {
		t.Error("Expected error")
	}
	if cb.State() != StateOpen {
		t.Errorf("Expected state Open after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreakerSuccessResetsFailures(t *testing.T) {
	config := Config{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
		MaxRetries:  0,
		RetryDelay:  0,
	}

	cb := New("test", config)

	// Some failures
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	// Success should reset failures
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}

	// Should still be closed
	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed, got %v", cb.State())
	}
}

func TestCircuitBreakerTimeout(t *testing.T) {
	config := Config{
		MaxFailures: 1,
		Timeout:     50 * time.Millisecond,
		MaxRetries:  0,
		RetryDelay:  0,
	}

	cb := New("test", config)

	// Execute something that takes too long
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := cb.Execute(ctx, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
}

func TestExecuteWithResult(t *testing.T) {
	config := DefaultConfig()
	cb := New("test", config)

	result, err := ExecuteWithResult(cb, context.Background(), func() (string, error) {
		return "success", nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %s", result)
	}

	// Test with error
	_, err = ExecuteWithResult(cb, context.Background(), func() (string, error) {
		return "", errors.New("failure")
	})
	if err == nil {
		t.Error("Expected error")
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	config := DefaultConfig()

	// Create new breaker
	cb1 := reg.GetOrCreate("s3", config)
	if cb1 == nil {
		t.Fatal("Expected non-nil breaker")
	}

	// Get existing breaker
	cb2 := reg.GetOrCreate("s3", config)
	if cb1 != cb2 {
		t.Error("Expected same breaker instance")
	}

	// Get method
	cb3, ok := reg.Get("s3")
	if !ok {
		t.Error("Expected to find breaker")
	}
	if cb1 != cb3 {
		t.Error("Expected same breaker instance")
	}

	// Get non-existent
	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("Expected not to find breaker")
	}

	// Stats
	stats := reg.AllStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 stat, got %d", len(stats))
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.state.String(); got != tc.expected {
			t.Errorf("State(%d).String() = %s, want %s", tc.state, got, tc.expected)
		}
	}
}
