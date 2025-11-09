package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_proxy_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "registry_proxy_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	// Cache Metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_proxy_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_proxy_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "registry_proxy_cache_size_bytes",
			Help: "Current cache size in bytes",
		},
		[]string{"cache_type"},
	)

	// Registry Metrics
	RegistryServersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "registry_proxy_servers_total",
			Help: "Total number of servers in registry",
		},
	)

	RegistryServersWithHeaders = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "registry_proxy_servers_with_headers_total",
			Help: "Number of servers with remote headers",
		},
	)

	UpstreamRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_proxy_upstream_requests_total",
			Help: "Total number of requests to upstream registry",
		},
		[]string{"endpoint", "status"},
	)

	UpstreamRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "registry_proxy_upstream_request_duration_seconds",
			Help:    "Upstream request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// Database Metrics
	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_proxy_database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "status"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "registry_proxy_database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "registry_proxy_database_connections_active",
			Help: "Number of active database connections",
		},
	)

	// Rating & Stats Metrics (FIXED: Removed high cardinality server_id labels)
	RatingsSubmittedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "registry_proxy_ratings_submitted_total",
			Help: "Total number of ratings submitted across all servers",
		},
	)

	InstallationsRecordedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "registry_proxy_installations_recorded_total",
			Help: "Total number of installations recorded across all servers",
		},
	)
)

// RecordHTTPRequest records an HTTP request with duration
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheType string) {
	CacheHitsTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheType string) {
	CacheMissesTotal.WithLabelValues(cacheType).Inc()
}

// RecordUpstreamRequest records an upstream registry request
func RecordUpstreamRequest(endpoint, status string, duration time.Duration) {
	UpstreamRequestsTotal.WithLabelValues(endpoint, status).Inc()
	UpstreamRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// RecordDatabaseQuery records a database query
func RecordDatabaseQuery(operation, status string, duration time.Duration) {
	DatabaseQueriesTotal.WithLabelValues(operation, status).Inc()
	DatabaseQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
