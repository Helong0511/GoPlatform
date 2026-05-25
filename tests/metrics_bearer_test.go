package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-platform/pkg/metrics"
)

func TestPrometheusMetricsHandlerOpenWhenNoToken(t *testing.T) {
	h := metrics.PrometheusMetricsHandler("")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without bearer requirement, got %d", w.Code)
	}
}

func TestPrometheusMetricsHandlerRequiresBearer(t *testing.T) {
	h := metrics.PrometheusMetricsHandler("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without Authorization, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req2.Header.Set("Authorization", "Bearer wrong")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", w2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req3.Header.Set("Authorization", "Bearer secret-token")
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid bearer, got %d", w3.Code)
	}
}
