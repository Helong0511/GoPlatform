package metrics

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-platform/pkg/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 监控指标结构体
type Metrics struct {
	httpRequestsTotal    prometheus.Counter
	httpRequestDuration  prometheus.Histogram
	dbQueriesTotal       prometheus.Counter
	dbQueryDuration      prometheus.Histogram
	kafkaMessagesTotal   prometheus.Counter
	redisOperationsTotal prometheus.Counter
}

var metricsLogger = logger.NewLogger("info")

// NewMetrics 创建监控指标
func NewMetrics() *Metrics {
	metrics := &Metrics{
		httpRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
			ConstLabels: prometheus.Labels{
				"service": "ad-platform",
			},
		}),
		httpRequestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds",
			ConstLabels: prometheus.Labels{
				"service": "ad-platform",
			},
			Buckets: prometheus.DefBuckets,
		}),
		dbQueriesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
			ConstLabels: prometheus.Labels{
				"service": "ad-platform",
			},
		}),
		dbQueryDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "db_query_duration_seconds",
			Help: "Database query duration in seconds",
			ConstLabels: prometheus.Labels{
				"service": "ad-platform",
			},
			Buckets: prometheus.DefBuckets,
		}),
		kafkaMessagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_total",
			Help: "Total number of Kafka messages",
			ConstLabels: prometheus.Labels{
				"service": "ad-platform",
			},
		}),
		redisOperationsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
			ConstLabels: prometheus.Labels{
				"service": "ad-platform",
			},
		}),
	}

	metrics.httpRequestsTotal = registerCounter(metrics.httpRequestsTotal)
	metrics.httpRequestDuration = registerHistogram(metrics.httpRequestDuration)
	metrics.dbQueriesTotal = registerCounter(metrics.dbQueriesTotal)
	metrics.dbQueryDuration = registerHistogram(metrics.dbQueryDuration)
	metrics.kafkaMessagesTotal = registerCounter(metrics.kafkaMessagesTotal)
	metrics.redisOperationsTotal = registerCounter(metrics.redisOperationsTotal)

	return metrics
}

// PrometheusMetricsHandler 返回 Prometheus 抓取 Handler；bearerToken 非空时要求 Authorization: Bearer <token>。
func PrometheusMetricsHandler(bearerToken string) http.Handler {
	inner := promhttp.Handler()
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return inner
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := bearerTokenFromAuthorization(r.Header.Get("Authorization"))
		if !constantTimeEqual(got, token) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="metrics"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		inner.ServeHTTP(w, r)
	})
}

func bearerTokenFromAuthorization(raw string) string {
	raw = strings.TrimSpace(raw)
	const prefix = "Bearer "
	if len(raw) >= len(prefix) && strings.EqualFold(raw[:len(prefix)], prefix) {
		return strings.TrimSpace(raw[len(prefix):])
	}
	return raw
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// StartServer 启动监控服务器（独立 HTTP）；host 为空时监听 0.0.0.0；返回 *http.Server 供进程退出时 Shutdown。
func StartServer(host string, port int, path string, bearerToken string) *http.Server {
	if path == "" {
		path = "/metrics"
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "0.0.0.0"
	}
	mux := http.NewServeMux()
	mux.Handle(path, PrometheusMetricsHandler(bearerToken))

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		metricsLogger.Info("Metrics server listening on %s:%d", host, port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			metricsLogger.Error("Failed to start metrics server: %v", err)
		}
	}()
	return server
}

// IncHTTPRequests 增加HTTP请求计数
func (m *Metrics) IncHTTPRequests() {
	m.httpRequestsTotal.Inc()
}

// ObserveHTTPRequestDuration 观察HTTP请求持续时间
func (m *Metrics) ObserveHTTPRequestDuration(duration float64) {
	m.httpRequestDuration.Observe(duration)
}

// IncDBQueries 增加数据库查询计数
func (m *Metrics) IncDBQueries() {
	m.dbQueriesTotal.Inc()
}

// ObserveDBQueryDuration 观察数据库查询持续时间
func (m *Metrics) ObserveDBQueryDuration(duration float64) {
	m.dbQueryDuration.Observe(duration)
}

// IncKafkaMessages 增加Kafka消息计数
func (m *Metrics) IncKafkaMessages() {
	m.kafkaMessagesTotal.Inc()
}

// IncRedisOperations 增加Redis操作计数
func (m *Metrics) IncRedisOperations() {
	m.redisOperationsTotal.Inc()
}

func registerCounter(counter prometheus.Counter) prometheus.Counter {
	err := prometheus.Register(counter)
	if err == nil {
		return counter
	}

	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	if errors.As(err, &alreadyRegisteredErr) {
		if existingCounter, ok := alreadyRegisteredErr.ExistingCollector.(prometheus.Counter); ok {
			return existingCounter
		}
		metricsLogger.Warn("Metrics counter registered with unexpected collector type: %v", err)
		return counter
	}

	metricsLogger.Error("Failed to register counter metric: %v", err)
	return counter
}

func registerHistogram(histogram prometheus.Histogram) prometheus.Histogram {
	err := prometheus.Register(histogram)
	if err == nil {
		return histogram
	}

	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	if errors.As(err, &alreadyRegisteredErr) {
		if existingHistogram, ok := alreadyRegisteredErr.ExistingCollector.(prometheus.Histogram); ok {
			return existingHistogram
		}
		metricsLogger.Warn("Metrics histogram registered with unexpected collector type: %v", err)
		return histogram
	}

	metricsLogger.Error("Failed to register histogram metric: %v", err)
	return histogram
}
