package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func TestBlockUnsafeHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.BlockUnsafeHTTPMethods())
	engine.Any("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, method := range []string{http.MethodTrace, "TRACK", http.MethodConnect} {
		req := httptest.NewRequest(method, "/ping", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("method %s: expected %d, got %d", method, http.StatusMethodNotAllowed, w.Code)
		}
	}

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
		req := httptest.NewRequest(method, "/ping", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("method %s: expected 200, got %d", method, w.Code)
		}
	}
}
