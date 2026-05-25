package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	apireport "ad-platform/api/report"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// 与 api/routes.go 一致：公开上报路由应挂载 API Key guard（此处用等价 stub 验证链式行为）。
func TestRouterStyleReportRequiresAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	public := engine.Group("/api")
	protected := engine.Group("/api/admin")

	handler := apireport.NewReportHandlerWithService(&fakeBatchReportService{})
	guard := func(c *gin.Context) {
		if c.GetHeader("X-API-Key") == "" {
			response.Unauthorized(c, response.MsgAPIKeyRequired)
			c.Abort()
			return
		}
		c.Next()
	}
	handler.SetupRoutes(public, protected, guard)

	body := []byte(`{"action":"click","user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API key, got %d", w.Code)
	}
}
