package util

import "strconv"

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// NormalizePagination 规范分页参数，避免非法值和过大页大小。
func NormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// ParsePaginationQuery 解析分页查询参数，并返回规范化后的分页值。
func ParsePaginationQuery(pageRaw, pageSizeRaw string) (int, int) {
	page, err := strconv.Atoi(pageRaw)
	if err != nil {
		page = DefaultPage
	}

	pageSize, err := strconv.Atoi(pageSizeRaw)
	if err != nil {
		pageSize = DefaultPageSize
	}

	return NormalizePagination(page, pageSize)
}
