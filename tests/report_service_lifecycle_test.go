package tests

import (
	"testing"
	"time"

	internalreport "ad-platform/internal/report"
)

func TestReportServiceCloseIsIdempotent(t *testing.T) {
	service := internalreport.NewReportService(nil, nil)

	done := make(chan struct{})
	go func() {
		service.Close()
		service.Close()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatalf("report service close should be idempotent and non-blocking")
	}
}

func TestReportServiceReportAdWithNilDBDoesNotPanic(t *testing.T) {
	service := internalreport.NewReportService(nil, nil)
	defer service.Close()

	report := &internalreport.AdReport{
		Action:    "click",
		UserID:    "u-1",
		Timestamp: time.Now().UTC(),
	}

	if err := service.ReportAd(report); err != nil {
		t.Fatalf("expected nil error when DB is unavailable fallback path is used, got %v", err)
	}
}
