package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	pkgauth "ad-platform/pkg/auth"

	"github.com/gin-gonic/gin"
)

func TestGetUserIDFromContextTypeMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", 12345)

	if _, ok := pkgauth.GetUserIDFromContext(c); ok {
		t.Fatalf("expected GetUserIDFromContext to return false when type is not string")
	}
}

func TestAuthMiddlewareRejectsNonBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authM := pkgauth.NewAuthMiddleware("test-secret")
	r := gin.New()
	r.Use(authM.AuthMiddleware())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "token-without-bearer")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
