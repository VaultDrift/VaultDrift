package tracing

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates OpenTelemetry tracing middleware for HTTP handlers.
type HTTPMiddleware struct {
	propagator propagation.TextMapPropagator
}

// NewHTTPMiddleware creates a new HTTP tracing middleware.
func NewHTTPMiddleware() *HTTPMiddleware {
	return &HTTPMiddleware{
		propagator: propagation.TraceContext{},
	}
}

// Middleware returns the HTTP tracing middleware handler.
func (m *HTTPMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming request
		ctx := m.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Create span for this request
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := Tracer("http-server").Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLScheme(r.URL.Scheme),
				semconv.URLPath(r.URL.Path),
				semconv.ServerAddress(r.Host),
				semconv.UserAgentOriginal(r.UserAgent()),
				semconv.HTTPRequestBodySize(int(r.ContentLength)),
				attribute.String("http.client_ip", getClientIP(r)),
			),
		)
		defer span.End()

		// Wrap response writer to capture status code
		wrapped := &tracingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add trace context to response headers
		m.propagator.Inject(ctx, propagation.HeaderCarrier(wrapped.Header()))

		// Record request start time
		start := time.Now()

		// Call next handler
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Record span attributes after request completes
		duration := time.Since(start)
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(wrapped.statusCode),
			attribute.Int64("http.response_size", wrapped.written),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		// Set status based on response code
		if wrapped.statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("Server error: %d", wrapped.statusCode))
		} else if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("Client error: %d", wrapped.statusCode))
		}
	})
}

// tracingResponseWriter wraps http.ResponseWriter to capture status and size.
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
	writtenHdr bool
}

func (w *tracingResponseWriter) WriteHeader(code int) {
	if w.writtenHdr {
		return
	}
	w.writtenHdr = true
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *tracingResponseWriter) Write(b []byte) (int, error) {
	if !w.writtenHdr {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-Ip header
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
