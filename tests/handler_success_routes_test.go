package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apireport "ad-platform/api/report"
	apiuser "ad-platform/api/user"
	"ad-platform/internal/admin"
	internalreport "ad-platform/internal/report"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

type fakeUserService struct {
	gotPage     int
	gotPageSize int
}

func (f *fakeUserService) ListUsers(page, pageSize int) ([]admin.User, int64, error) {
	f.gotPage = page
	f.gotPageSize = pageSize
	return []admin.User{
		{ID: 1, Username: "u1"},
		{ID: 2, Username: "u2"},
	}, 2, nil
}

func (f *fakeUserService) GetUserByID(id uint) (*admin.User, error) { return &admin.User{ID: id}, nil }
func (f *fakeUserService) CreateUser(user *admin.User) error        { return nil }
func (f *fakeUserService) UpdateUser(user *admin.User) error        { return nil }
func (f *fakeUserService) DeleteUser(id uint) error                 { return nil }
func (f *fakeUserService) EnableUser(id, operatorID uint, ip string) error {
	return nil
}
func (f *fakeUserService) DisableUser(id, operatorID uint, ip string) error {
	return nil
}
func (f *fakeUserService) GetUserStatusLogs(userID uint, page, pageSize int) ([]admin.UserStatusLog, int64, error) {
	return []admin.UserStatusLog{}, 0, nil
}

type fakeReportService struct {
	gotPage     int
	gotPageSize int
	gotFilters  map[string]interface{}
}

func (f *fakeReportService) ReportAd(report *internalreport.AdReport) error { return nil }
func (f *fakeReportService) GetTodayReportsByTimezone(timezone string) (int64, error) {
	return 0, nil
}
func (f *fakeReportService) GetReportStatsByTimezone(timezone string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (f *fakeReportService) ListAdReportsWithFilter(page, pageSize int, filters map[string]interface{}) ([]internalreport.AdReport, int64, error) {
	f.gotPage = page
	f.gotPageSize = pageSize
	f.gotFilters = filters
	return []internalreport.AdReport{
		{ID: 10, AdID: 12, Action: "click", Timestamp: time.Now().UTC()},
	}, 21, nil
}

func TestUserListRouteSuccessWithMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/api/admin")
	protected.Use(injectPermissions("user:read"))

	fakeSvc := &fakeUserService{}
	handler := apiuser.NewUserHandlerWithService(fakeSvc)
	handler.SetupRoutes(protected)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users?page=-3&page_size=999", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, resp.Code)
	}
	if resp.Meta == nil {
		t.Fatalf("expected meta in paginated response")
	}
	if resp.Meta.Page != 1 || resp.Meta.PageSize != 100 || resp.Meta.Total != 2 || resp.Meta.TotalPages != 1 {
		t.Fatalf("unexpected pagination meta: %+v", resp.Meta)
	}
	if fakeSvc.gotPage != 1 || fakeSvc.gotPageSize != 100 {
		t.Fatalf("expected normalized pagination passed to service, got page=%d pageSize=%d", fakeSvc.gotPage, fakeSvc.gotPageSize)
	}
}

func TestReportListRouteSuccessWithMetaAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	public := engine.Group("/api")
	protected := engine.Group("/api/admin")
	protected.Use(injectPermissions("report:read"))

	fakeSvc := &fakeReportService{}
	handler := apireport.NewReportHandlerWithService(fakeSvc)
	handler.SetupRoutes(public, protected)

	start := "2026-04-20T00:00:00Z"
	end := "2026-04-20T02:00:00Z"
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/admin/reports?page=2&page_size=20&ad_id=12&action=click&start_time="+start+"&end_time="+end,
		nil,
	)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, resp.Code)
	}
	if resp.Meta == nil {
		t.Fatalf("expected meta in paginated response")
	}
	if resp.Meta.Page != 2 || resp.Meta.PageSize != 20 || resp.Meta.Total != 21 || resp.Meta.TotalPages != 2 {
		t.Fatalf("unexpected pagination meta: %+v", resp.Meta)
	}
	if fakeSvc.gotPage != 2 || fakeSvc.gotPageSize != 20 {
		t.Fatalf("expected page=2,pageSize=20 passed to service, got page=%d pageSize=%d", fakeSvc.gotPage, fakeSvc.gotPageSize)
	}
	if _, ok := fakeSvc.gotFilters["ad_id"].(uint); !ok {
		t.Fatalf("expected ad_id filter to be uint, got %#v", fakeSvc.gotFilters["ad_id"])
	}
	if action, ok := fakeSvc.gotFilters["action"].(string); !ok || action != "click" {
		t.Fatalf("expected action=click filter, got %#v", fakeSvc.gotFilters["action"])
	}
	startTime, ok := fakeSvc.gotFilters["start_time"].(time.Time)
	if !ok {
		t.Fatalf("expected start_time filter to be time.Time")
	}
	endTime, ok := fakeSvc.gotFilters["end_time"].(time.Time)
	if !ok {
		t.Fatalf("expected end_time filter to be time.Time")
	}
	if !startTime.Before(endTime) {
		t.Fatalf("expected start_time before end_time, got start=%s end=%s", startTime, endTime)
	}
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return resp
}
