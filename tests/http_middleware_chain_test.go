package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-platform/pkg/logger"
	"ad-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// 与 cmd/main.go 全局中间件顺序一致（简化版，无 CORS/限流）。
func TestMainMiddlewareChainOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(
		logger.RecoveryMiddleware(),
		middleware.BlockUnsafeHTTPMethods(),
		middleware.MaxRequestBodyBytes(16),
		middleware.SecurityHeadersMiddleware(0, false),
	)
	engine.POST("/echo", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("CONNECT blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodConnect, "/echo", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("CONNECT: got %d", w.Code)
		}
	})

	t.Run("oversize body 413", func(t *testing.T) {
		body := bytes.Repeat([]byte("x"), 32)
		req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(body))
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversize: got %d", w.Code)
		}
	})

	t.Run("security headers present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte("ok")))
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("missing X-Content-Type-Options")
		}
	})
}
