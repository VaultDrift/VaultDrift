package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// RecoveryMiddleware recovers from panics and returns a 500 error.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v\n%s", err, debug.Stack())
				http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		// Sanitize path to prevent log injection
		path := strings.ReplaceAll(r.URL.Path, "\n", "")
		path = strings.ReplaceAll(path, "\r", "")
		log.Printf("%s %s %s %d %s", // #nosec G706 - path is sanitized above
			r.Method,
			path,
			r.RemoteAddr,
			wrapped.statusCode,
			duration,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.written {
		return
	}
	rw.written = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// CORSMiddleware handles Cross-Origin Resource Sharing.
func CORSMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if an origin is in the allowed list.
func isOriginAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return true // Allow non-browser requests
	}

	// If no restrictions, allow all
	if len(allowed) == 0 || (len(allowed) == 1 && allowed[0] == "*") {
		return true
	}

	for _, allowed := range allowed {
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			suffix := allowed[1:] // Remove *
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}

	return false
}

// SecurityHeadersMiddleware adds security headers to responses.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
		// HSTS - only set if request is HTTPS (detected via X-Forwarded-Proto header or TLS state)
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware adds a request ID to each request.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add to response header
		w.Header().Set("X-Request-ID", requestID)

		// Add to context for use in handlers
		ctx := WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a simple request ID.
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// RateLimitMiddleware simple rate limiting middleware.
type RateLimitMiddleware struct {
	requests   map[string][]time.Time
	mu         sync.RWMutex
	limit      int
	window     time.Duration
	lastClean  time.Time
}

// NewRateLimitMiddleware creates a new rate limiter.
func NewRateLimitMiddleware(limit int, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requests:  make(map[string][]time.Time),
		limit:     limit,
		window:    window,
		lastClean: time.Now(),
	}
}

// Limit returns the rate limiting middleware handler.
func (rl *RateLimitMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.RemoteAddr // Could use API key or user ID for better tracking

		now := time.Now()
		cutoff := now.Add(-rl.window)

		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Periodic cleanup of old clients (every 100 requests or 1 minute)
		if now.Sub(rl.lastClean) > time.Minute {
			rl.cleanupOldClients(now)
			rl.lastClean = now
		}

		// Clean old requests and count current
		var count int
		for _, t := range rl.requests[clientID] {
			if t.After(cutoff) {
				count++
			}
		}

		if count >= rl.limit {
			http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// Add current request
		rl.requests[clientID] = append(rl.requests[clientID], now)

		next.ServeHTTP(w, r)
	})
}

// cleanupOldClients removes clients with no recent requests to prevent memory leak.
func (rl *RateLimitMiddleware) cleanupOldClients(now time.Time) {
	cutoff := now.Add(-rl.window)
	for clientID, requests := range rl.requests {
		hasRecent := false
		for _, t := range requests {
			if t.After(cutoff) {
				hasRecent = true
				break
			}
		}
		if !hasRecent {
			delete(rl.requests, clientID)
		}
	}
}
