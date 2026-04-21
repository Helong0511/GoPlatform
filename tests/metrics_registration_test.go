package tests

import (
	"testing"

	"ad-platform/pkg/metrics"
)

func TestNewMetricsCanBeCalledMultipleTimes(t *testing.T) {
	var first *metrics.Metrics
	var second *metrics.Metrics

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("NewMetrics should not panic when called repeatedly, panic: %v", r)
			}
		}()
		first = metrics.NewMetrics()
		second = metrics.NewMetrics()
	}()

	if first == nil || second == nil {
		t.Fatalf("expected both metrics instances to be non-nil")
	}

	// 调用核心方法确保注册后的collector可正常工作
	first.IncHTTPRequests()
	first.ObserveHTTPRequestDuration(0.1)
	first.IncDBQueries()
	first.ObserveDBQueryDuration(0.2)
	first.IncKafkaMessages()
	first.IncRedisOperations()

	second.IncHTTPRequests()
	second.ObserveHTTPRequestDuration(0.3)
	second.IncDBQueries()
	second.ObserveDBQueryDuration(0.4)
	second.IncKafkaMessages()
	second.IncRedisOperations()
}
