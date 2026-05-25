package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apisystem "ad-platform/api/system"

	"github.com/gin-gonic/gin"
)

func TestHealthReadyIncludesVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := apisystem.NewSystemHandler(false, func(context.Context) error { return nil }, nil, nil)
	handler.SetupRoutes(engine)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	v, ok := body["version"].(string)
	if !ok || v == "" {
		t.Fatalf("expected non-empty version string in ready response, got %#v", body["version"])
	}
}
