package tests

import (
	"testing"
	"time"

	"ad-platform/pkg/util"
)

func TestParseOptionalUintQuery(t *testing.T) {
	value, ok, detail := util.ParseOptionalUintQuery("ad_id", "")
	if ok || detail != nil || value != 0 {
		t.Fatalf("expected empty value to be ignored")
	}

	value, ok, detail = util.ParseOptionalUintQuery("ad_id", "123")
	if !ok || detail != nil || value != 123 {
		t.Fatalf("expected valid ad_id to be parsed")
	}

	_, ok, detail = util.ParseOptionalUintQuery("ad_id", "abc")
	if ok || detail == nil {
		t.Fatalf("expected invalid ad_id to return validation detail")
	}
}

func TestParseOptionalRFC3339Query(t *testing.T) {
	parsed, ok, detail := util.ParseOptionalRFC3339Query("start_time", "")
	if ok || detail != nil || !parsed.IsZero() {
		t.Fatalf("expected empty time value to be ignored")
	}

	parsed, ok, detail = util.ParseOptionalRFC3339Query("start_time", "2026-04-20T08:00:00+08:00")
	if !ok || detail != nil {
		t.Fatalf("expected valid RFC3339 time to be parsed")
	}
	if parsed.Format(time.RFC3339) != "2026-04-20T00:00:00Z" {
		t.Fatalf("expected parsed time to be normalized to UTC, got %s", parsed.Format(time.RFC3339))
	}

	_, ok, detail = util.ParseOptionalRFC3339Query("start_time", "2026/04/20 08:00:00")
	if ok || detail == nil {
		t.Fatalf("expected invalid time format to return validation detail")
	}
}

func TestValidateSortParams(t *testing.T) {
	allowedFields := map[string]struct{}{
		"id":         {},
		"name":       {},
		"created_at": {},
	}

	details := util.ValidateSortParams("name", "desc", allowedFields)
	if len(details) != 0 {
		t.Fatalf("expected valid sort params to pass")
	}

	details = util.ValidateSortParams("priority", "invalid", allowedFields)
	if len(details) != 2 {
		t.Fatalf("expected two validation errors, got %d", len(details))
	}
}

func TestValidateTimeRange(t *testing.T) {
	start := time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)

	if detail := util.ValidateTimeRange(start, end); detail != nil {
		t.Fatalf("expected valid time range")
	}

	if detail := util.ValidateTimeRange(end, start); detail == nil {
		t.Fatalf("expected invalid time range to return validation detail")
	}
}
