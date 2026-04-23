package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiad "ad-platform/api/ad"
	apireport "ad-platform/api/report"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestAdActiveRouteValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	public := engine.Group("/api")
	protected := engine.Group("/api/admin")
	protected.Use(injectPermissions("ad:read"))
	handler := apiad.NewAdHandler(nil, nil)
	handler.SetupRoutes(engine, public, protected)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/admin/ads/active?sort_field=bad_field&sort_order=up&start_time=not-a-time",
		nil,
	)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Code != response.CodeValidationError {
		t.Fatalf("expected code %d, got %d", response.CodeValidationError, resp.Code)
	}

	assertHasErrorField(t, resp.Errors, "sort_field")
	assertHasErrorField(t, resp.Errors, "sort_order")
	assertHasErrorField(t, resp.Errors, "start_time")
}

func TestReportListRouteValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	public := engine.Group("/api")
	protected := engine.Group("/api/admin")
	protected.Use(injectPermissions("report:read"))
	handler := apireport.NewReportHandler(nil, nil)
	handler.SetupRoutes(engine, public, protected)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/admin/reports?ad_id=abc&start_time=2026-04-20T10:00:00Z&end_time=2026-04-20T09:00:00Z",
		nil,
	)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Code != response.CodeValidationError {
		t.Fatalf("expected code %d, got %d", response.CodeValidationError, resp.Code)
	}

	assertHasErrorField(t, resp.Errors, "ad_id")
	assertHasErrorField(t, resp.Errors, "end_time")
}

func assertHasErrorField(t *testing.T, details []response.ErrorDetail, field string) {
	t.Helper()
	for _, detail := range details {
		if detail.Field == field {
			return
		}
	}
	t.Fatalf("expected validation error field %q, got %+v", field, details)
}
