package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-platform/pkg/logger"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddlewareGeneratesHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(logger.RequestIDMiddleware())
	engine.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}
	if got := w.Header().Get(logger.RequestIDHeader); got == "" {
		t.Fatalf("expected response header %s", logger.RequestIDHeader)
	}
}

func TestRequestIDMiddlewareKeepsIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(logger.RequestIDMiddleware())
	engine.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	const incoming = "req-abc-123"
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(logger.RequestIDHeader, incoming)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}
	if got := w.Header().Get(logger.RequestIDHeader); got != incoming {
		t.Fatalf("expected request id %q, got %q", incoming, got)
	}
}
