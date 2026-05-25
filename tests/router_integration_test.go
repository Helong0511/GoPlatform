package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-platform/api"
	"ad-platform/internal/ad"
	"ad-platform/internal/admin"
	"ad-platform/internal/report"
	"ad-platform/internal/security"
	"ad-platform/pkg/db"
	"ad-platform/pkg/redis"

	"github.com/gin-gonic/gin"
)

// 验证 api.NewRouter 挂载的公开上报路由与生产一致：无 API Key 时 401。
func TestNewRouterPublicReportRequiresAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc := admin.NewUserService(&db.PostgresDB{}, redis.NewMockRedisClient(), 8)
	roleSvc := admin.NewRoleService(&db.PostgresDB{})
	deptSvc := admin.NewDepartmentService(&db.PostgresDB{})
	adSvc := ad.NewAdService(&db.PostgresDB{}, redis.NewMockRedisClient())
	reportSvc := report.NewReportService(&db.PostgresDB{}, nil)
	secMgr, err := security.NewSecurityManager("")
	if err != nil {
		t.Fatalf("security manager: %v", err)
	}

	router := api.NewRouter(
		userSvc, roleSvc, deptSvc, adSvc, reportSvc, secMgr,
		"integration-test-jwt-secret-32chars-long",
		"ad-platform", "ad-platform-api",
		time.Hour,
		false,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	engine := gin.New()
	router.SetupRoutes(engine)

	body := []byte(`{"action":"click","user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API key, got %d body=%s", w.Code, w.Body.String())
	}
}
