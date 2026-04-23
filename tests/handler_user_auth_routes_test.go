package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiuser "ad-platform/api/user"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestUserEnableDisableAuthValidationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("enable user with invalid id returns bad request", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("user:update"))
		handler := apiuser.NewUserHandler(nil)
		handler.SetupRoutes(protected)

		req := httptest.NewRequest(http.MethodPut, "/api/admin/users/abc/enable", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assertResponseCode(t, w, http.StatusBadRequest, response.CodeBadRequest)
	})

	t.Run("enable user without auth context returns unauthorized", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		handler := apiuser.NewUserHandler(nil)
		handler.SetupRoutes(protected)

		req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/enable", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assertResponseCode(t, w, http.StatusUnauthorized, response.CodeUnauthorized)
	})

	t.Run("enable user with invalid operator id returns unauthorized", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("user:update"))
		protected.Use(func(c *gin.Context) {
			c.Set("user_id", "not-a-number")
			c.Next()
		})
		handler := apiuser.NewUserHandler(nil)
		handler.SetupRoutes(protected)

		req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/enable", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assertResponseCode(t, w, http.StatusUnauthorized, response.CodeUnauthorized)
	})

	t.Run("disable user without auth context returns unauthorized", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		handler := apiuser.NewUserHandler(nil)
		handler.SetupRoutes(protected)

		req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/disable", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assertResponseCode(t, w, http.StatusUnauthorized, response.CodeUnauthorized)
	})

	t.Run("disable user with invalid operator id returns unauthorized", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("user:update"))
		protected.Use(func(c *gin.Context) {
			c.Set("user_id", "not-a-number")
			c.Next()
		})
		handler := apiuser.NewUserHandler(nil)
		handler.SetupRoutes(protected)

		req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/disable", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assertResponseCode(t, w, http.StatusUnauthorized, response.CodeUnauthorized)
	})
}

func assertResponseCode(t *testing.T, recorder *httptest.ResponseRecorder, expectedHTTPStatus int, expectedCode response.ResponseCode) {
	t.Helper()

	if recorder.Code != expectedHTTPStatus {
		t.Fatalf("expected status %d, got %d", expectedHTTPStatus, recorder.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Code != expectedCode {
		t.Fatalf("expected code %d, got %d", expectedCode, resp.Code)
	}
}
