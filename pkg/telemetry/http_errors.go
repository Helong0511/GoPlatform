package telemetry

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

var httpErrorsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP error responses grouped by reason/category/code",
		ConstLabels: prometheus.Labels{
			"service": "ad-platform",
		},
	},
	[]string{"error_code", "error_category", "error_reason"},
)

func init() {
	err := prometheus.Register(httpErrorsTotal)
	if err == nil {
		return
	}
	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	if errors.As(err, &alreadyRegisteredErr) {
		if collector, ok := alreadyRegisteredErr.ExistingCollector.(*prometheus.CounterVec); ok {
			httpErrorsTotal = collector
		}
	}
}

// RecordHTTPError 记录HTTP错误指标。
func RecordHTTPError(code string, category string, reason string) {
	if code == "" {
		code = "unknown"
	}
	if category == "" {
		category = "unknown"
	}
	if reason == "" {
		reason = "unknown"
	}
	httpErrorsTotal.WithLabelValues(code, category, reason).Inc()
}
