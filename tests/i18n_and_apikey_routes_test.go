package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apisystem "ad-platform/api/system"
	apiuser "ad-platform/api/user"
	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

type fakeAPIKeyUserService struct{}

func (f *fakeAPIKeyUserService) ListUsers(page, pageSize int) ([]admin.User, int64, error) {
	return []admin.User{}, 0, nil
}
func (f *fakeAPIKeyUserService) GetUserByID(id uint) (*admin.User, error) { return &admin.User{ID: id}, nil }
func (f *fakeAPIKeyUserService) CreateUser(user *admin.User) error        { return nil }
func (f *fakeAPIKeyUserService) UpdateUser(user *admin.User) error        { return nil }
func (f *fakeAPIKeyUserService) DeleteUser(id uint) error                 { return nil }
func (f *fakeAPIKeyUserService) EnableUser(id, operatorID uint, ip string) error {
	return nil
}
func (f *fakeAPIKeyUserService) DisableUser(id, operatorID uint, ip string) error {
	return nil
}
func (f *fakeAPIKeyUserService) GetUserStatusLogs(userID uint, page, pageSize int) ([]admin.UserStatusLog, int64, error) {
	return []admin.UserStatusLog{}, 0, nil
}
func (f *fakeAPIKeyUserService) ListUserAPIKeys(userID uint) ([]admin.UserAPIKey, error) {
	return []admin.UserAPIKey{}, nil
}
func (f *fakeAPIKeyUserService) DisableUserAPIKey(userID, keyID uint) error { return nil }
func (f *fakeAPIKeyUserService) RotateUserAPIKey(userID uint, name, scope string, expiresAt *time.Time) (string, error) {
	if scope == "bad:scope" {
		return "", common.ErrInvalidArgument
	}
	return "new-api-key", nil
}

type fakeAPIKeyNotFoundUserService struct{ fakeAPIKeyUserService }

func (f *fakeAPIKeyNotFoundUserService) DisableUserAPIKey(userID, keyID uint) error {
	return common.ErrAPIKeyNotFound
}

func TestRotateAPIKeyInvalidJSONAndScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid json returns bad request", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("user:update"))

		handler := apiuser.NewUserHandlerWithService(&fakeAPIKeyUserService{})
		handler.SetupRoutes(protected)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/users/1/api-keys/rotate", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("invalid scope returns bad request", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("user:update"))

		handler := apiuser.NewUserHandlerWithService(&fakeAPIKeyUserService{})
		handler.SetupRoutes(protected)

		body := []byte(`{"scope":"bad:scope"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/users/1/api-keys/rotate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var resp response.Response
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response body: %v", err)
		}
		if resp.Code != response.CodeBadRequest {
			t.Fatalf("expected code %d, got %d", response.CodeBadRequest, resp.Code)
		}
	})

	t.Run("expires_in_days too large returns bad request", func(t *testing.T) {
		engine := gin.New()
		protected := engine.Group("/api/admin")
		protected.Use(injectPermissions("user:update"))

		handler := apiuser.NewUserHandlerWithService(&fakeAPIKeyUserService{})
		handler.SetupRoutes(protected)

		body := []byte(`{"expires_in_days":3651}`)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/users/1/api-keys/rotate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestDisableAPIKeyNotFoundReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/api/admin")
	protected.Use(injectPermissions("user:update"))

	handler := apiuser.NewUserHandlerWithService(&fakeAPIKeyNotFoundUserService{})
	handler.SetupRoutes(protected)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/api-keys/123/disable", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSystemI18NMissingAndResetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 先清理，避免受其它测试影响
	response.ResetMissingTranslations()
	_ = response.Translate(nil, "totally-new-message")
	_ = response.Translate(nil, "another-new-message")

	engine := gin.New()
	protected := engine.Group("/api/admin")
	protected.Use(injectPermissions("user:read", "user:update"))

	handler := apisystem.NewSystemHandler(false)
	handler.SetupAdminRoutes(protected)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/system/i18n-missing", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var getResp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data, ok := getResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be object")
	}
	missing, ok := data["missing"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected missing object in response")
	}
	zhList, ok := missing["zh-CN"].([]interface{})
	if !ok || len(zhList) == 0 {
		t.Fatalf("expected zh-CN missing list to be non-empty")
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/api/admin/system/i18n-missing/reset", nil)
	resetW := httptest.NewRecorder()
	engine.ServeHTTP(resetW, resetReq)
	if resetW.Code != http.StatusOK {
		t.Fatalf("expected reset status %d, got %d", http.StatusOK, resetW.Code)
	}

	checkReq := httptest.NewRequest(http.MethodGet, "/api/admin/system/i18n-missing", nil)
	checkW := httptest.NewRecorder()
	engine.ServeHTTP(checkW, checkReq)
	if checkW.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, checkW.Code)
	}

	var checkResp response.Response
	if err := json.Unmarshal(checkW.Body.Bytes(), &checkResp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	checkData, ok := checkResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be object")
	}
	checkMissing, ok := checkData["missing"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected missing object in response")
	}
	checkZhList, ok := checkMissing["zh-CN"].([]interface{})
	if !ok {
		t.Fatalf("expected zh-CN missing list to exist")
	}
	if len(checkZhList) != 0 {
		t.Fatalf("expected zh-CN missing list to be empty after reset")
	}
}

