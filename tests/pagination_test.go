package tests

import (
	"testing"

	"ad-platform/pkg/util"
)

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name             string
		page             int
		pageSize         int
		expectedPage     int
		expectedPageSize int
	}{
		{
			name:             "negative page and pageSize use defaults",
			page:             -1,
			pageSize:         -10,
			expectedPage:     1,
			expectedPageSize: 10,
		},
		{
			name:             "zero page and pageSize use defaults",
			page:             0,
			pageSize:         0,
			expectedPage:     1,
			expectedPageSize: 10,
		},
		{
			name:             "pageSize over max is clamped",
			page:             2,
			pageSize:         150,
			expectedPage:     2,
			expectedPageSize: 100,
		},
		{
			name:             "valid values keep original",
			page:             3,
			pageSize:         25,
			expectedPage:     3,
			expectedPageSize: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, pageSize := util.NormalizePagination(tt.page, tt.pageSize)
			if page != tt.expectedPage {
				t.Fatalf("expected page %d, got %d", tt.expectedPage, page)
			}
			if pageSize != tt.expectedPageSize {
				t.Fatalf("expected pageSize %d, got %d", tt.expectedPageSize, pageSize)
			}
		})
	}
}

func TestParsePaginationQuery(t *testing.T) {
	tests := []struct {
		name             string
		pageRaw          string
		pageSizeRaw      string
		expectedPage     int
		expectedPageSize int
	}{
		{
			name:             "invalid query values fallback to defaults",
			pageRaw:          "abc",
			pageSizeRaw:      "xyz",
			expectedPage:     1,
			expectedPageSize: 10,
		},
		{
			name:             "valid page with too large pageSize is clamped",
			pageRaw:          "5",
			pageSizeRaw:      "999",
			expectedPage:     5,
			expectedPageSize: 100,
		},
		{
			name:             "negative page and zero pageSize are normalized",
			pageRaw:          "-3",
			pageSizeRaw:      "0",
			expectedPage:     1,
			expectedPageSize: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, pageSize := util.ParsePaginationQuery(tt.pageRaw, tt.pageSizeRaw)
			if page != tt.expectedPage {
				t.Fatalf("expected page %d, got %d", tt.expectedPage, page)
			}
			if pageSize != tt.expectedPageSize {
				t.Fatalf("expected pageSize %d, got %d", tt.expectedPageSize, pageSize)
			}
		})
	}
}
