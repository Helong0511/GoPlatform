package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiauth "ad-platform/api/auth"
	apireport "ad-platform/api/report"
	apisystem "ad-platform/api/system"
	internalreport "ad-platform/internal/report"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

type fakeBatchReportService struct{}

func (f *fakeBatchReportService) ReportAd(report *internalreport.AdReport) error { return nil }
func (f *fakeBatchReportService) GetTodayReportsByTimezone(timezone string) (int64, error) {
	return 0, nil
}
func (f *fakeBatchReportService) GetReportStatsByTimezone(timezone string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (f *fakeBatchReportService) ListAdReportsWithFilter(page, pageSize int, filters map[string]interface{}) ([]internalreport.AdReport, int64, error) {
	return []internalreport.AdReport{}, 0, nil
}

func TestRegisterDisabledReturnsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	public := engine.Group("/api")
	protected := engine.Group("/api/admin")
	handler := apiauth.NewAuthHandler(nil, "test-secret", "ad-platform", "ad-platform-api", 0, false)
	handler.SetupRoutes(public, protected, nil, nil)

	body := []byte(`{"username":"u1","password":"123456","email":"u1@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected code %d, got %d", response.CodeForbidden, resp.Code)
	}
}

func TestReportBatchRoutePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing report:create returns forbidden", func(t *testing.T) {
		engine := gin.New()
		public := engine.Group("/api")
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("report:read"))

		handler := apireport.NewReportHandlerWithService(&fakeBatchReportService{})
		handler.SetupRoutes(public, protected)

		body := []byte(`{"reports":[{"action":"click"}]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/reports/batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("report:create can batch create", func(t *testing.T) {
		engine := gin.New()
		public := engine.Group("/api")
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("report:create"))

		handler := apireport.NewReportHandlerWithService(&fakeBatchReportService{})
		handler.SetupRoutes(public, protected)

		body := []byte(`{"reports":[{"action":"click"}]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/reports/batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestReportPublicRouteRequiresAPIKeyWhenGuardProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	public := engine.Group("/api")
	protected := engine.Group("/api/admin")

	handler := apireport.NewReportHandlerWithService(&fakeBatchReportService{})
	guard := func(c *gin.Context) {
		response.Unauthorized(c, response.MsgAPIKeyRequired)
		c.Abort()
	}
	handler.SetupRoutes(public, protected, guard)

	body := []byte(`{"action":"click","user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d without api key, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestSystemRootEndpointRegisterVisibility(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("register hidden when disabled", func(t *testing.T) {
		engine := gin.New()
		handler := apisystem.NewSystemHandler(false, nil, nil, nil)
		handler.SetupRoutes(engine)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response body: %v", err)
		}
		endpoints, ok := resp["endpoints"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected endpoints field in root response")
		}
		if _, exists := endpoints["register"]; exists {
			t.Fatalf("register endpoint should not be exposed when disabled")
		}
	})

	t.Run("register shown when enabled", func(t *testing.T) {
		engine := gin.New()
		handler := apisystem.NewSystemHandler(true, nil, nil, nil)
		handler.SetupRoutes(engine)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response body: %v", err)
		}
		endpoints, ok := resp["endpoints"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected endpoints field in root response")
		}
		if endpoints["register"] != "/api/register" {
			t.Fatalf("expected register endpoint to be exposed when enabled")
		}
	})
}
