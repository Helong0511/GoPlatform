package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func TestMaxRequestBodyBytesRejectsOversize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.MaxRequestBodyBytes(32))
	r.POST("/echo", func(c *gin.Context) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(c.Request.Body)
		c.String(http.StatusOK, "ok")
	})

	body := bytes.Repeat([]byte("a"), 64)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
	}
}

func TestSecurityHeadersHSTSOnlyOverHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rPlain := gin.New()
	rPlain.Use(middleware.SecurityHeadersMiddleware(3600, false))
	rPlain.GET("/x", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	rPlain.ServeHTTP(w, req)
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("expected no HSTS when forwarded proto is not trusted")
	}

	rTrusted := gin.New()
	rTrusted.Use(middleware.SecurityHeadersMiddleware(3600, true))
	rTrusted.GET("/x", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.Header.Set("X-Forwarded-Proto", "https")
	w2 := httptest.NewRecorder()
	rTrusted.ServeHTTP(w2, req2)
	h := w2.Header().Get("Strict-Transport-Security")
	if !strings.Contains(h, "max-age=3600") || !strings.Contains(h, "includeSubDomains") {
		t.Fatalf("unexpected HSTS header: %q", h)
	}
}

func TestClientScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "HTTPS")
	if middleware.ClientScheme(c, true) != "https" {
		t.Fatalf("expected https from forwarded proto when trusted")
	}
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	c2.Request.Header.Set("X-Forwarded-Proto", "HTTPS")
	if middleware.ClientScheme(c2, false) != "http" {
		t.Fatalf("expected http when forwarded proto is not trusted")
	}
}
