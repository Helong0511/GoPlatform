package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ad-platform/pkg/response"
)

// ParseOptionalUintQuery 解析可选的无符号整型查询参数。
func ParseOptionalUintQuery(field, raw string) (uint, bool, *response.ErrorDetail) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, false, &response.ErrorDetail{
			Field:   field,
			Message: fmt.Sprintf("%s must be a valid positive integer", field),
		}
	}

	return uint(parsed), true, nil
}

// ParseOptionalRFC3339Query 解析可选的 RFC3339 时间查询参数。
func ParseOptionalRFC3339Query(field, raw string) (time.Time, bool, *response.ErrorDetail) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false, &response.ErrorDetail{
			Field:   field,
			Message: fmt.Sprintf("%s must be RFC3339 format", field),
		}
	}

	return parsed.UTC(), true, nil
}

// ValidateSortParams 校验排序字段及顺序。
func ValidateSortParams(sortField, sortOrder string, allowedFields map[string]struct{}) []response.ErrorDetail {
	var details []response.ErrorDetail

	if _, ok := allowedFields[sortField]; !ok {
		details = append(details, response.ErrorDetail{
			Field:   "sort_field",
			Message: "sort_field is not allowed",
		})
	}

	order := strings.ToLower(sortOrder)
	if order != "asc" && order != "desc" {
		details = append(details, response.ErrorDetail{
			Field:   "sort_order",
			Message: "sort_order must be asc or desc",
		})
	}

	return details
}

// ValidateTimeRange 校验时间范围是否有效。
func ValidateTimeRange(start, end time.Time) *response.ErrorDetail {
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return &response.ErrorDetail{
			Field:   "end_time",
			Message: "end_time must be greater than or equal to start_time",
		}
	}
	return nil
}
