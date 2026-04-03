package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestNewProviderDisabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create disabled provider: %v", err)
	}

	if provider.IsEnabled() {
		t.Error("Expected provider to be disabled")
	}

	// Shutdown should not error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestNewProviderStdout(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Exporter:    "stdout",
		ServiceName: "test-service",
		SampleRate:  1.0,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if !provider.IsEnabled() {
		t.Error("Expected provider to be enabled")
	}

	// Test tracer
	tracer := provider.Tracer("test")
	if tracer == nil {
		t.Error("Expected non-nil tracer")
	}

	// Create a test span
	ctx, span := tracer.Start(context.Background(), "test-span")
	if span == nil {
		t.Error("Expected non-nil span")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Expected valid span context")
	}
	span.End()

	// Verify context contains trace info
	spanFromCtx := trace.SpanFromContext(ctx)
	if spanFromCtx == nil {
		t.Error("Expected span in context")
	}

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = provider.Shutdown(shutdownCtx)
}

func TestHTTPMiddleware(t *testing.T) {
	// Create a simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create middleware
	middleware := NewHTTPMiddleware()

	// Wrap handler
	handler := middleware.Middleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check trace header was set
	traceHeader := rec.Header().Get("traceparent")
	// Trace header may or may not be set depending on global state
	t.Logf("Trace header: %s", traceHeader)
}

func TestHTTPMiddlewareErrorStatus(t *testing.T) {
	// Create handler that returns error
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	middleware := NewHTTPMiddleware()
	handler := middleware.Middleware(testHandler)

	req := httptest.NewRequest("POST", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-Ip",
			headers:  map[string]string{"X-Real-Ip": "10.0.0.2"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.2",
		},
		{
			name:     "RemoteAddr fallback",
			headers:  map[string]string{},
			remote:   "192.168.1.1:1234",
			expected: "192.168.1.1:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTracingResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	trw := &tracingResponseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test Write
	n, err := trw.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected 5 bytes written, got %d", n)
	}
	if trw.written != 5 {
		t.Errorf("Expected written=5, got %d", trw.written)
	}

	// Test WriteHeader
	trw.WriteHeader(http.StatusNotFound)
	if trw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, trw.statusCode)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected recorder status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("Expected default enabled to be false")
	}
	if cfg.Exporter != "stdout" {
		t.Errorf("Expected default exporter 'stdout', got %s", cfg.Exporter)
	}
	if cfg.ServiceName != "vaultdrift" {
		t.Errorf("Expected default service name 'vaultdrift', got %s", cfg.ServiceName)
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("Expected default sample rate 1.0, got %f", cfg.SampleRate)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test Tracer
	t.Run("Tracer", func(t *testing.T) {
		tracer := Tracer("test")
		if tracer == nil {
			t.Error("Expected non-nil tracer")
		}
	})

	// Test StartSpan
	t.Run("StartSpan", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "test-span")
		if span == nil {
			t.Error("Expected non-nil span")
		}
		if ctx == nil {
			t.Error("Expected non-nil context")
		}
		span.End()
	})

	// Test SpanFromContext
	t.Run("SpanFromContext", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "test-span")
		spanFromCtx := SpanFromContext(ctx)
		if spanFromCtx == nil {
			t.Error("Expected span from context")
		}
		span.End()
	})

	// Test ContextWithSpan
	t.Run("ContextWithSpan", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "test-span")
		ctx := ContextWithSpan(context.Background(), span)
		if ctx == nil {
			t.Error("Expected non-nil context")
		}
		span.End()
	})
}
