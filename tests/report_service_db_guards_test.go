package tests

import (
	"testing"

	internalreport "ad-platform/internal/report"
)

func TestReportServiceDBGuardMethodsReturnErrorWhenDBUnavailable(t *testing.T) {
	service := internalreport.NewReportService(nil, nil)
	defer service.Close()

	testCases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "GetTotalReports",
			fn: func() error {
				_, err := service.GetTotalReports()
				return err
			},
		},
		{
			name: "GetTotalClicks",
			fn: func() error {
				_, err := service.GetTotalClicks()
				return err
			},
		},
		{
			name: "GetTodayReports",
			fn: func() error {
				_, err := service.GetTodayReports()
				return err
			},
		},
		{
			name: "GetTodayReportsByTimezone",
			fn: func() error {
				_, err := service.GetTodayReportsByTimezone("Asia/Shanghai")
				return err
			},
		},
		{
			name: "GetReportStatsByTimezone",
			fn: func() error {
				_, err := service.GetReportStatsByTimezone("Asia/Shanghai")
				return err
			},
		},
		{
			name: "ListAdReports",
			fn: func() error {
				_, _, err := service.ListAdReports(1, 10)
				return err
			},
		},
		{
			name: "ListAdReportsWithFilter",
			fn: func() error {
				_, _, err := service.ListAdReportsWithFilter(1, 10, map[string]interface{}{"action": "click"})
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Fatalf("expected error when report DB is unavailable")
			}
		})
	}
}
