package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics는 애플리케이션 메트릭을 관리합니다
type Metrics struct {
	// HTTP 메트릭
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// gRPC 메트릭
	GRPCRequestsTotal   *prometheus.CounterVec
	GRPCRequestDuration *prometheus.HistogramVec

	// 데이터베이스 메트릭
	DBOperationsTotal   *prometheus.CounterVec
	DBOperationDuration *prometheus.HistogramVec
	DBConnectionsActive prometheus.Gauge

	// 캐시 메트릭
	CacheHitsTotal   *prometheus.CounterVec
	CacheMissesTotal *prometheus.CounterVec

	// 시스템 메트릭
	GoroutinesActive prometheus.Gauge
}

var globalMetrics *Metrics

// Init은 메트릭을 초기화합니다
func Init(namespace string) *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		GRPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "grpc_requests_total",
				Help:      "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),
		GRPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "grpc_request_duration_seconds",
				Help:      "gRPC request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method"},
		),
		DBOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_operations_total",
				Help:      "Total number of database operations",
			},
			[]string{"operation", "collection", "status"},
		),
		DBOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_operation_duration_seconds",
				Help:      "Database operation duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "collection"},
		),
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_active",
				Help:      "Number of active database connections",
			},
		),
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_name"},
		),
		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_name"},
		),
		GoroutinesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "goroutines_active",
				Help:      "Number of active goroutines",
			},
		),
	}

	globalMetrics = m
	return m
}

// GetMetrics는 글로벌 메트릭 인스턴스를 반환합니다
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		return Init("database_service")
	}
	return globalMetrics
}

// RecordHTTPRequest는 HTTP 요청 메트릭을 기록합니다
func (m *Metrics) RecordHTTPRequest(method, endpoint, status string, duration time.Duration, requestSize, responseSize int) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordGRPCRequest는 gRPC 요청 메트릭을 기록합니다
func (m *Metrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	m.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordDBOperation은 데이터베이스 작업 메트릭을 기록합니다
func (m *Metrics) RecordDBOperation(operation, collection, status string, duration time.Duration) {
	m.DBOperationsTotal.WithLabelValues(operation, collection, status).Inc()
	m.DBOperationDuration.WithLabelValues(operation, collection).Observe(duration.Seconds())
}

// RecordCacheHit은 캐시 히트를 기록합니다
func (m *Metrics) RecordCacheHit(cacheName string) {
	m.CacheHitsTotal.WithLabelValues(cacheName).Inc()
}

// RecordCacheMiss는 캐시 미스를 기록합니다
func (m *Metrics) RecordCacheMiss(cacheName string) {
	m.CacheMissesTotal.WithLabelValues(cacheName).Inc()
}
