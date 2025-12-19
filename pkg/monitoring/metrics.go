package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector manages application metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Agent metrics
	agentTotal          prometheus.Gauge
	agentActive         prometheus.Gauge
	agentMessageTotal   *prometheus.CounterVec
	agentErrorTotal     *prometheus.CounterVec
	agentSessionTotal   *prometheus.CounterVec
	agentTokenUsage     *prometheus.CounterVec

	// LLM metrics
	llmRequestsTotal    *prometheus.CounterVec
	llmRequestDuration  *prometheus.HistogramVec
	llmTokenUsage       *prometheus.CounterVec
	llmErrorsTotal      *prometheus.CounterVec

	// System metrics
	systemMemoryUsage   prometheus.Gauge
	systemCPUUsage      prometheus.Gauge
	systemGoroutineCount prometheus.Gauge

	// Custom metrics
	customMetrics map[string]prometheus.Metric
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	collector := &MetricsCollector{
		customMetrics: make(map[string]prometheus.Metric),
	}

	collector.initMetrics()
	return collector
}

// initMetrics initializes all Prometheus metrics
func (m *MetricsCollector) initMetrics() {
	// HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	m.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	// Agent metrics
	m.agentTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "agent_total",
			Help: "Total number of agents",
		},
	)

	m.agentActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "agent_active",
			Help: "Number of active agents",
		},
	)

	m.agentMessageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_messages_total",
			Help: "Total number of agent messages",
		},
		[]string{"session_id", "role"},
	)

	m.agentErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_errors_total",
			Help: "Total number of agent errors",
		},
		[]string{"session_id", "error_type"},
	)

	m.agentSessionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_sessions_total",
			Help: "Total number of agent sessions",
		},
		[]string{"action"},
	)

	m.agentTokenUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_token_usage_total",
			Help: "Total token usage",
		},
		[]string{"session_id", "type"},
	)

	// LLM metrics
	m.llmRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total number of LLM requests",
		},
		[]string{"provider", "model", "status"},
	)

	m.llmRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "LLM request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "model"},
	)

	m.llmTokenUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_token_usage_total",
			Help: "Total LLM token usage",
		},
		[]string{"provider", "model", "type"},
	)

	m.llmErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_errors_total",
			Help: "Total number of LLM errors",
		},
		[]string{"provider", "model", "error_type"},
	)

	// System metrics
	m.systemMemoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "System memory usage in bytes",
		},
	)

	m.systemCPUUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "System CPU usage percentage",
		},
	)

	m.systemGoroutineCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_goroutine_count",
			Help: "Number of goroutines",
		},
	)

	// Register all metrics with Prometheus
	prometheus.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestSize,
		m.httpResponseSize,
		m.agentTotal,
		m.agentActive,
		m.agentMessageTotal,
		m.agentErrorTotal,
		m.agentSessionTotal,
		m.agentTokenUsage,
		m.llmRequestsTotal,
		m.llmRequestDuration,
		m.llmTokenUsage,
		m.llmErrorsTotal,
		m.systemMemoryUsage,
		m.systemCPUUsage,
		m.systemGoroutineCount,
	)
}

// HTTP Metrics Methods

// RecordHTTPRequest records an HTTP request
func (m *MetricsCollector) RecordHTTPRequest(method, endpoint, status string, duration time.Duration, requestSize, responseSize int64) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	m.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// Agent Metrics Methods

// SetAgentCount sets the total number of agents
func (m *MetricsCollector) SetAgentCount(total, active int) {
	m.agentTotal.Set(float64(total))
	m.agentActive.Set(float64(active))
}

// RecordAgentMessage records an agent message
func (m *MetricsCollector) RecordAgentMessage(sessionID, role string) {
	m.agentMessageTotal.WithLabelValues(sessionID, role).Inc()
}

// RecordAgentError records an agent error
func (m *MetricsCollector) RecordAgentError(sessionID, errorType string) {
	m.agentErrorTotal.WithLabelValues(sessionID, errorType).Inc()
}

// RecordAgentSession records an agent session event
func (m *MetricsCollector) RecordAgentSession(action string) {
	m.agentSessionTotal.WithLabelValues(action).Inc()
}

// RecordAgentTokenUsage records token usage
func (m *MetricsCollector) RecordAgentTokenUsage(sessionID, tokenType string, count int64) {
	m.agentTokenUsage.WithLabelValues(sessionID, tokenType).Add(float64(count))
}

// LLM Metrics Methods

// RecordLLMRequest records an LLM request
func (m *MetricsCollector) RecordLLMRequest(provider, model, status string, duration time.Duration) {
	m.llmRequestsTotal.WithLabelValues(provider, model, status).Inc()
	m.llmRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordLLMTokenUsage records LLM token usage
func (m *MetricsCollector) RecordLLMTokenUsage(provider, model, tokenType string, count int64) {
	m.llmTokenUsage.WithLabelValues(provider, model, tokenType).Add(float64(count))
}

// RecordLLMError records an LLM error
func (m *MetricsCollector) RecordLLMError(provider, model, errorType string) {
	m.llmErrorsTotal.WithLabelValues(provider, model, errorType).Inc()
}

// System Metrics Methods

// UpdateSystemMetrics updates system-level metrics
func (m *MetricsCollector) UpdateSystemMetrics() {
	// This would typically collect actual system metrics
	// For now, we'll just update the goroutine count
	m.systemGoroutineCount.Set(float64(getGoroutineCount()))
}

// getGoroutineCount returns the current number of goroutines
func getGoroutineCount() int {
	// This is a placeholder implementation
	// In a real implementation, you'd use runtime.GoroutineProfile or similar
	return 0
}

// Custom Metrics Methods

// RegisterCustomMetric registers a custom metric
func (m *MetricsCollector) RegisterCustomMetric(name string, metric prometheus.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.customMetrics[name]; exists {
		return fmt.Errorf("metric %s already exists", name)
	}

	m.customMetrics[name] = metric
	prometheus.MustRegister(metric.(prometheus.Collector))
	return nil
}

// GetCustomMetric retrieves a custom metric
func (m *MetricsCollector) GetCustomMetric(name string) (prometheus.Metric, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metric, exists := m.customMetrics[name]
	return metric, exists
}

// MetricsServer serves metrics via HTTP
type MetricsServer struct {
	collector *MetricsCollector
	server    *http.Server
	port      int
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(collector *MetricsCollector, port int) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &MetricsServer{
		collector: collector,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		port: port,
	}
}

// Start starts the metrics server
func (ms *MetricsServer) Start() error {
	return ms.server.ListenAndServe()
}

// Stop stops the metrics server
func (ms *MetricsServer) Stop(ctx context.Context) error {
	return ms.server.Shutdown(ctx)
}

// HealthChecker performs health checks
type HealthChecker struct {
	checks map[string]HealthCheck
	mu     sync.RWMutex
}

// HealthCheck represents a health check function
type HealthCheck func(ctx context.Context) error

// HealthStatus represents the status of a health check
type HealthStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "healthy", "unhealthy", "unknown"
	Message   string    `json:"message"`
	LastCheck time.Time `json:"last_check"`
	Duration  time.Duration `json:"duration"`
	Error     string    `json:"error,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]HealthCheck),
	}
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// CheckHealth performs all registered health checks
func (hc *HealthChecker) CheckHealth(ctx context.Context) map[string]HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]HealthStatus)

	for name, check := range hc.checks {
		status := HealthStatus{
			Name:      name,
			Status:    "unknown",
			LastCheck: time.Now(),
		}

		start := time.Now()
		err := check(ctx)
		status.Duration = time.Since(start)

		if err != nil {
			status.Status = "unhealthy"
			status.Message = "Health check failed"
			status.Error = err.Error()
		} else {
			status.Status = "healthy"
			status.Message = "Health check passed"
		}

		results[name] = status
	}

	return results
}