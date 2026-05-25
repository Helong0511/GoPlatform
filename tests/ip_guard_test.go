package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestIPGuardAutoBanOnReportUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	guard := middleware.NewIPGuard(middleware.IPGuardOptions{
		Enabled:          true,
		BanDuration:      time.Hour,
		StrikeWindow:     time.Minute,
		StrikesBeforeBan: 3,
	}, nil)

	engine := gin.New()
	engine.Use(guard.Middleware())
	engine.POST("/api/report", func(c *gin.Context) {
		response.SetErrorReason(c, response.ReasonMissingAPIKey)
		response.Unauthorized(c, response.MsgAPIKeyRequired)
	})

	const clientIP = "203.0.113.50"
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/report", nil)
		req.RemoteAddr = clientIP + ":12345"
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/report", nil)
	req.RemoteAddr = clientIP + ":12345"
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("after ban: expected 403, got %d", w.Code)
	}
}

func TestIPGuardCrawlerStrikeOnAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	guard := middleware.NewIPGuard(middleware.IPGuardOptions{
		Enabled:               true,
		BanDuration:           time.Hour,
		StrikeWindow:          time.Minute,
		StrikesBeforeBan:      3,
		BlockCrawlerUserAgent: true,
		CrawlerStrikeWeight:   2,
	}, nil)

	engine := gin.New()
	engine.Use(guard.Middleware())
	engine.GET("/api/admin/users", func(c *gin.Context) {
		response.SetErrorReason(c, response.ReasonUserNotAuthenticated)
		response.Unauthorized(c, response.MsgUserNotAuthenticated)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.RemoteAddr = "198.51.100.10:12345"
	req.Header.Set("User-Agent", "python-requests/2.31")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestIPGuardWhitelistSkipsBan(t *testing.T) {
	guard := middleware.NewIPGuard(middleware.IPGuardOptions{
		Enabled:          true,
		BanDuration:      time.Hour,
		StrikeWindow:     time.Minute,
		StrikesBeforeBan: 1,
	}, nil)
	if err := guard.AddWhitelist("10.0.0.9", "office"); err != nil {
		t.Fatal(err)
	}
	if err := guard.Ban("10.0.0.9", "should fail"); err == nil {
		t.Fatal("expected ban on whitelisted ip to fail")
	}
}

func TestIPGuardUnban(t *testing.T) {
	guard := middleware.NewIPGuard(middleware.IPGuardOptions{
		Enabled:          true,
		BanDuration:      time.Hour,
		StrikeWindow:     time.Minute,
		StrikesBeforeBan: 1,
	}, nil)
	guard.Ban("10.0.0.1", "test")
	if !guard.Unban("10.0.0.1") {
		t.Fatal("expected unban to succeed")
	}
	for _, e := range guard.ListBanned() {
		if e.IP == "10.0.0.1" {
			t.Fatal("ip should be removed after unban")
		}
	}
}
