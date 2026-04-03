package server

import (
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// Metrics holds application metrics for monitoring.
type Metrics struct {
	db *db.Manager

	// HTTP metrics
	requestsTotal   *expvar.Int
	requestsActive  int64 // atomic
	requestsErrors  *expvar.Int
	requestDuration *expvar.Map

	// Storage metrics
	bytesUploaded   *expvar.Int
	bytesDownloaded *expvar.Int
	chunksTotal     *expvar.Int
	chunksDeduped   *expvar.Int

	// Auth metrics
	loginsTotal     *expvar.Int
	loginsFailed    *expvar.Int
	tokensIssued    *expvar.Int
	tokensRefreshed *expvar.Int

	// WebSocket metrics
	wsConnections    int64 // atomic
	wsConnectionsMax int64 // atomic
	wsMessagesSent   *expvar.Int
	wsMessagesRecv   *expvar.Int

	// System metrics (last values)
	lastGCStats     atomic.Value
	lastHealthCheck atomic.Value
}

// safeNewInt creates an expvar.Int only if it doesn't exist.
func safeNewInt(name string) *expvar.Int {
	v := expvar.Get(name)
	if v != nil {
		return v.(*expvar.Int)
	}
	return expvar.NewInt(name)
}

// safeNewMap creates an expvar.Map only if it doesn't exist.
func safeNewMap(name string) *expvar.Map {
	v := expvar.Get(name)
	if v != nil {
		return v.(*expvar.Map)
	}
	return expvar.NewMap(name)
}

// NewMetrics creates a new metrics collector.
func NewMetrics(database *db.Manager) *Metrics {
	m := &Metrics{
		db: database,

		// HTTP metrics
		requestsTotal:   safeNewInt("http_requests_total"),
		requestsErrors:  safeNewInt("http_requests_errors_total"),
		requestDuration: safeNewMap("http_request_duration_ms"),

		// Storage metrics
		bytesUploaded:   safeNewInt("storage_bytes_uploaded_total"),
		bytesDownloaded: safeNewInt("storage_bytes_downloaded_total"),
		chunksTotal:     safeNewInt("storage_chunks_total"),
		chunksDeduped:   safeNewInt("storage_chunks_deduped_total"),

		// Auth metrics
		loginsTotal:     safeNewInt("auth_logins_total"),
		loginsFailed:    safeNewInt("auth_logins_failed_total"),
		tokensIssued:    safeNewInt("auth_tokens_issued_total"),
		tokensRefreshed: safeNewInt("auth_tokens_refreshed_total"),

		// WebSocket metrics
		wsMessagesSent: safeNewInt("websocket_messages_sent_total"),
		wsMessagesRecv: safeNewInt("websocket_messages_received_total"),
	}

	// Initialize last values
	m.lastHealthCheck.Store(time.Now())
	m.lastGCStats.Store(&runtime.MemStats{})

	return m
}

// RecordRequest records an HTTP request metric.
func (m *Metrics) RecordRequest(method, path string, duration time.Duration, statusCode int) {
	m.requestsTotal.Add(1)

	// Bucket by status code family
	key := fmt.Sprintf("%s_%s_%dxx", method, path, statusCode/100)
	m.requestDuration.Add(key, duration.Milliseconds())

	if statusCode >= 400 {
		m.requestsErrors.Add(1)
	}
}

// RecordBytesUploaded records uploaded bytes.
func (m *Metrics) RecordBytesUploaded(n int64) {
	m.bytesUploaded.Add(n)
}

// RecordBytesDownloaded records downloaded bytes.
func (m *Metrics) RecordBytesDownloaded(n int64) {
	m.bytesDownloaded.Add(n)
}

// RecordChunk records chunk processing.
func (m *Metrics) RecordChunk(deduplicated bool) {
	m.chunksTotal.Add(1)
	if deduplicated {
		m.chunksDeduped.Add(1)
	}
}

// RecordLogin records login attempt.
func (m *Metrics) RecordLogin(success bool) {
	m.loginsTotal.Add(1)
	if !success {
		m.loginsFailed.Add(1)
	}
}

// RecordToken records token operations.
func (m *Metrics) RecordToken(refresh bool) {
	if refresh {
		m.tokensRefreshed.Add(1)
	} else {
		m.tokensIssued.Add(1)
	}
}

// RecordWSConnection records WebSocket connection.
func (m *Metrics) RecordWSConnection(active bool) {
	if active {
		current := atomic.AddInt64(&m.wsConnections, 1)
		// Track max connections
		for {
			max := atomic.LoadInt64(&m.wsConnectionsMax)
			if current <= max {
				break
			}
			if atomic.CompareAndSwapInt64(&m.wsConnectionsMax, max, current) {
				break
			}
		}
	} else {
		atomic.AddInt64(&m.wsConnections, -1)
	}
}

// RecordWSMessage records WebSocket message.
func (m *Metrics) RecordWSMessage(sent bool) {
	if sent {
		m.wsMessagesSent.Add(1)
	} else {
		m.wsMessagesRecv.Add(1)
	}
}

// ActiveRequests returns current active request count.
func (m *Metrics) ActiveRequests() int64 {
	return atomic.LoadInt64(&m.requestsActive)
}

// IncActiveRequest increments active request counter.
func (m *Metrics) IncActiveRequest() {
	atomic.AddInt64(&m.requestsActive, 1)
}

// DecActiveRequest decrements active request counter.
func (m *Metrics) DecActiveRequest() {
	atomic.AddInt64(&m.requestsActive, -1)
}

// WSConnections returns current WebSocket connection count.
func (m *Metrics) WSConnections() int64 {
	return atomic.LoadInt64(&m.wsConnections)
}

// WSConnectionsMax returns max WebSocket connection count.
func (m *Metrics) WSConnectionsMax() int64 {
	return atomic.LoadInt64(&m.wsConnectionsMax)
}

// MetricsHandler returns HTTP handler for metrics endpoint.
func (m *Metrics) MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add runtime metrics
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		// Export as JSON
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "{\n")
		fmt.Fprintf(w, "  \"timestamp\": %d,\n", time.Now().Unix())
		fmt.Fprintf(w, "  \"version\": \"0.1.0\",\n")

		// Runtime metrics
		fmt.Fprintf(w, "  \"runtime\": {\n")
		fmt.Fprintf(w, "    \"goroutines\": %d,\n", runtime.NumGoroutine())
		fmt.Fprintf(w, "    \"memory\": {\n")
		fmt.Fprintf(w, "      \"alloc_bytes\": %d,\n", memStats.Alloc)
		fmt.Fprintf(w, "      \"total_alloc_bytes\": %d,\n", memStats.TotalAlloc)
		fmt.Fprintf(w, "      \"sys_bytes\": %d,\n", memStats.Sys)
		fmt.Fprintf(w, "      \"heap_alloc_bytes\": %d,\n", memStats.HeapAlloc)
		fmt.Fprintf(w, "      \"heap_sys_bytes\": %d,\n", memStats.HeapSys)
		fmt.Fprintf(w, "      \"heap_inuse_bytes\": %d,\n", memStats.HeapInuse)
		fmt.Fprintf(w, "      \"gc_count\": %d,\n", memStats.NumGC)
		fmt.Fprintf(w, "      \"gc_pause_ns\": %d\n", memStats.PauseNs[(memStats.NumGC+255)%256])
		fmt.Fprintf(w, "    }\n")
		fmt.Fprintf(w, "  },\n")

		// Database metrics
		if m.db != nil {
			stats := m.db.Stats()
			fmt.Fprintf(w, "  \"database\": {\n")
			fmt.Fprintf(w, "    \"max_open_connections\": %d,\n", stats.MaxOpenConnections)
			fmt.Fprintf(w, "    \"open_connections\": %d,\n", stats.OpenConnections)
			fmt.Fprintf(w, "    \"in_use\": %d,\n", stats.InUse)
			fmt.Fprintf(w, "    \"idle\": %d,\n", stats.Idle)
			fmt.Fprintf(w, "    \"wait_count\": %d,\n", stats.WaitCount)
			fmt.Fprintf(w, "    \"wait_duration_ms\": %d\n", stats.WaitDuration.Milliseconds())
			fmt.Fprintf(w, "  },\n")
		}

		// Application metrics
		fmt.Fprintf(w, "  \"application\": {\n")
		fmt.Fprintf(w, "    \"http_requests_total\": %s,\n", m.requestsTotal.String())
		fmt.Fprintf(w, "    \"http_requests_active\": %d,\n", m.ActiveRequests())
		fmt.Fprintf(w, "    \"http_requests_errors_total\": %s,\n", m.requestsErrors.String())
		fmt.Fprintf(w, "    \"storage_bytes_uploaded_total\": %s,\n", m.bytesUploaded.String())
		fmt.Fprintf(w, "    \"storage_bytes_downloaded_total\": %s,\n", m.bytesDownloaded.String())
		fmt.Fprintf(w, "    \"auth_logins_total\": %s,\n", m.loginsTotal.String())
		fmt.Fprintf(w, "    \"auth_logins_failed_total\": %s,\n", m.loginsFailed.String())
		fmt.Fprintf(w, "    \"websocket_connections_active\": %d,\n", m.WSConnections())
		fmt.Fprintf(w, "    \"websocket_connections_max\": %d\n", m.WSConnectionsMax())
		fmt.Fprintf(w, "  }\n")

		fmt.Fprintf(w, "}\n")
	})
}

// PrometheusHandler returns metrics in Prometheus text format.
func (m *Metrics) PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		// Helper to write metric
		writeMetric := func(name, help, typ string, value int64) {
			fmt.Fprintf(w, "# HELP %s %s\n", name, help)
			fmt.Fprintf(w, "# TYPE %s %s\n", name, typ)
			fmt.Fprintf(w, "%s %d\n", name, value)
		}

		// Runtime metrics
		writeMetric("go_goroutines", "Number of goroutines", "gauge", int64(runtime.NumGoroutine()))
		writeMetric("go_memory_alloc_bytes", "Bytes allocated", "gauge", int64(memStats.Alloc))
		writeMetric("go_memory_sys_bytes", "Bytes obtained from system", "gauge", int64(memStats.Sys))
		writeMetric("go_gc_count", "Number of GC cycles", "counter", int64(memStats.NumGC))

		// HTTP metrics
		writeMetric("vaultdrift_http_requests_total", "Total HTTP requests", "counter", m.requestsTotal.Value())
		writeMetric("vaultdrift_http_requests_active", "Active HTTP requests", "gauge", m.ActiveRequests())
		writeMetric("vaultdrift_http_requests_errors_total", "Total HTTP errors", "counter", m.requestsErrors.Value())

		// Storage metrics
		writeMetric("vaultdrift_storage_bytes_uploaded_total", "Total bytes uploaded", "counter", m.bytesUploaded.Value())
		writeMetric("vaultdrift_storage_bytes_downloaded_total", "Total bytes downloaded", "counter", m.bytesDownloaded.Value())
		writeMetric("vaultdrift_storage_chunks_total", "Total chunks processed", "counter", m.chunksTotal.Value())
		writeMetric("vaultdrift_storage_chunks_deduped_total", "Deduplicated chunks", "counter", m.chunksDeduped.Value())

		// Auth metrics
		writeMetric("vaultdrift_auth_logins_total", "Total login attempts", "counter", m.loginsTotal.Value())
		writeMetric("vaultdrift_auth_logins_failed_total", "Failed login attempts", "counter", m.loginsFailed.Value())
		writeMetric("vaultdrift_auth_tokens_issued_total", "Tokens issued", "counter", m.tokensIssued.Value())
		writeMetric("vaultdrift_auth_tokens_refreshed_total", "Tokens refreshed", "counter", m.tokensRefreshed.Value())

		// WebSocket metrics
		writeMetric("vaultdrift_websocket_connections_active", "Active WebSocket connections", "gauge", m.WSConnections())
		writeMetric("vaultdrift_websocket_connections_max", "Max concurrent WebSocket connections", "gauge", m.WSConnectionsMax())
		writeMetric("vaultdrift_websocket_messages_sent_total", "WebSocket messages sent", "counter", m.wsMessagesSent.Value())
		writeMetric("vaultdrift_websocket_messages_received_total", "WebSocket messages received", "counter", m.wsMessagesRecv.Value())

		// Database metrics
		if m.db != nil {
			stats := m.db.Stats()
			writeMetric("vaultdrift_db_connections_open", "Open database connections", "gauge", int64(stats.OpenConnections))
			writeMetric("vaultdrift_db_connections_in_use", "In-use database connections", "gauge", int64(stats.InUse))
			writeMetric("vaultdrift_db_connections_idle", "Idle database connections", "gauge", int64(stats.Idle))
			writeMetric("vaultdrift_db_wait_count_total", "Total connection waits", "counter", int64(stats.WaitCount))
		}
	})
}

// ParseBool parses boolean from string (for feature flags).
func ParseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
