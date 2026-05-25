package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	authM := pkgauth.NewAuthMiddleware("test-secret", "ad-platform", "ad-platform-api")
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

func TestGenerateJWTRoundTripMultiRoles(t *testing.T) {
	mw := pkgauth.NewAuthMiddleware("test-secret-roles", "ad-platform", "ad-platform-api")
	tok, err := mw.GenerateJWT("1001", "Asia/Shanghai", []string{"admin", "analyst", "analyst"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	claims, err := mw.ValidateJWT(tok)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}
	if claims.UserID != "1001" || claims.Timezone != "Asia/Shanghai" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.Role != "admin" {
		t.Fatalf("expected primary role admin, got %q", claims.Role)
	}
	if len(claims.Roles) != 3 {
		t.Fatalf("expected 3 roles in token (duplicate preserved), got %d %+v", len(claims.Roles), claims.Roles)
	}
}

func TestAuthMiddlewareSetsRolesInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := pkgauth.NewAuthMiddleware("ctx-secret", "ad-platform", "ad-platform-api")
	tok, err := mw.GenerateJWT("7", "UTC", []string{"r1", "r2"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	r := gin.New()
	r.Use(mw.AuthMiddleware())
	r.GET("/p", func(c *gin.Context) {
		roles, ok := pkgauth.GetRolesFromContext(c)
		if !ok || len(roles) != 2 || roles[0] != "r1" || roles[1] != "r2" {
			t.Errorf("GetRolesFromContext = (%v, %v)", roles, ok)
		}
		primary, ok := pkgauth.GetRoleFromContext(c)
		if !ok || primary != "r1" {
			t.Errorf("GetRoleFromContext = (%q, %v)", primary, ok)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
