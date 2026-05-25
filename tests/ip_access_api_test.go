package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-platform/api"
	"ad-platform/api/system"
	"ad-platform/internal/ad"
	"ad-platform/internal/admin"
	"ad-platform/internal/report"
	"ad-platform/internal/security"
	"ad-platform/pkg/db"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/redis"

	"github.com/gin-gonic/gin"
)

func TestIPAccessAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guard := middleware.NewIPGuard(middleware.IPGuardOptions{
		Enabled:          true,
		BanDuration:      time.Hour,
		StrikeWindow:     time.Minute,
		StrikesBeforeBan: 5,
	}, nil)

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
		false, nil, nil, nil, nil, guard, nil,
	)

	engine := gin.New()
	// 模拟已认证管理员（仅测 system 路由挂载；RBAC 在完整栈中由 UserContext 提供）
	protected := engine.Group("/api/admin")
	sys := system.NewSystemHandler(false, nil, guard, nil)
	sys.SetupAdminRoutes(protected)

	t.Run("status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/system/ip-access/status", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusOK {
			// 无 JWT 时可能 401；有中间件时 401 说明路由存在
			if w.Code == http.StatusNotFound {
				t.Fatalf("route not mounted, got %d", w.Code)
			}
		}
	})

	t.Run("blacklist CRUD without auth returns non-404", func(t *testing.T) {
		for _, path := range []string{
			"/api/admin/system/ip-blacklist",
			"/api/admin/system/ip-whitelist",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code == http.StatusNotFound {
				t.Fatalf("%s not found", path)
			}
		}
	})

	t.Run("ban and list via guard directly", func(t *testing.T) {
		if err := guard.Ban("192.0.2.100", "test"); err != nil {
			t.Fatal(err)
		}
		entries := guard.ListBanned()
		found := false
		for _, e := range entries {
			if e.IP == "192.0.2.100" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected banned ip in list")
		}
		if !guard.Unban("192.0.2.100") {
			t.Fatal("unban failed")
		}
	})

	t.Run("whitelist add removes ban", func(t *testing.T) {
		_ = guard.Ban("192.0.2.200", "x")
		if err := guard.AddWhitelist("192.0.2.200", "trusted"); err != nil {
			t.Fatal(err)
		}
		if err := guard.Ban("192.0.2.200", "again"); err == nil {
			t.Fatal("expected cannot ban whitelisted ip")
		}
		_ = guard.RemoveWhitelist("192.0.2.200")
	})

	_ = router // 完整路由栈由 api.NewRouter 提供，此处仅直连 system 子路由做冒烟
}

func TestParseIPGuardOptionsFromYAMLFields(t *testing.T) {
	opts, err := middleware.ParseIPGuardOptions(middleware.IPBlacklistYAML{
		Enabled:            true,
		BanDuration:        "24h",
		StrikeWindow:       "10m",
		StrikesBeforeBan:   5,
		StaticIPs:          []string{"192.0.2.1"},
		StaticWhitelistIPs: []string{"192.0.2.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.BanDuration != 24*time.Hour || opts.StrikeWindow != 10*time.Minute {
		t.Fatalf("unexpected durations: ban=%v window=%v", opts.BanDuration, opts.StrikeWindow)
	}
	guard := middleware.NewIPGuard(opts, nil)
	if !guard.IsWhitelisted("192.0.2.2") {
		t.Fatal("static whitelist ip should be whitelisted")
	}
	found := false
	for _, e := range guard.ListBanned() {
		if e.IP == "192.0.2.1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("static blacklist ip should be banned")
	}
}

func TestIPBlacklistDeletePathParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	guard := middleware.NewIPGuard(middleware.IPGuardOptions{Enabled: true}, nil)
	_ = guard.Ban("10.1.2.3", "manual")

	engine := gin.New()
	protected := engine.Group("/api/admin")
	sys := system.NewSystemHandler(false, nil, guard, nil)
	sys.SetupAdminRoutes(protected)

	// 路由存在性：无 Bearer 时通常 401，不应 404
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/system/ip-blacklist/10.1.2.3", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Fatalf("delete route missing, body=%s", w.Body.String())
	}

	body, _ := json.Marshal(map[string]string{"ip": "192.0.2.50", "reason": "scan"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/system/ip-blacklist", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	engine.ServeHTTP(w2, req2)
	if w2.Code == http.StatusNotFound {
		t.Fatalf("post ban route missing")
	}
}
