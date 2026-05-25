// Package response 统一API响应包
// 提供标准化的API响应格式、错误码体系和响应辅助函数
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// ContextResponseCodeKey 用于在 gin 上下文中透传业务响应码，便于统一日志记录。
	ContextResponseCodeKey = "response_code"
	// ContextResponseMessageKey 用于在 gin 上下文中透传响应消息，便于统一日志记录。
	ContextResponseMessageKey = "response_message"
	// ContextErrorCategoryKey 用于在 gin 上下文中透传错误分类，便于日志聚合。
	ContextErrorCategoryKey = "error_category"
	// ContextErrorReasonKey 用于在 gin 上下文中透传错误原因，便于排障定位。
	ContextErrorReasonKey = "error_reason"
)

const (
	ReasonMissingAuthHeader      = "missing_auth_header"
	ReasonInvalidBearerFormat    = "invalid_bearer_format"
	ReasonMissingBearerToken     = "missing_bearer_token"
	ReasonInvalidOrExpiredToken  = "invalid_or_expired_token"
	ReasonUserNotAuthenticated   = "user_not_authenticated"
	ReasonInvalidUserPrincipal   = "invalid_user_principal"
	ReasonUserLocked             = "user_locked"
	ReasonInsufficientPermission = "insufficient_permission"
	ReasonDepartmentNotAssigned  = "department_not_assigned"
	ReasonMissingAPIKey          = "missing_api_key"
	ReasonInvalidAPIKey          = "invalid_api_key"
	ReasonAPIKeyScopeDenied      = "api_key_scope_denied"
	ReasonRateLimited            = "rate_limited"
	ReasonCircuitOpen            = "circuit_breaker_open"
	ReasonIPBlacklisted          = "ip_blacklisted"
)

// ResponseCode 响应状态码类型
type ResponseCode int

// 错误码定义
const (
	// 成功码
	CodeSuccess ResponseCode = 200

	// 客户端错误 (4xx)
	CodeBadRequest       ResponseCode = 400 // 请求参数错误
	CodeUnauthorized     ResponseCode = 401 // 未授权
	CodeForbidden        ResponseCode = 403 // 禁止访问
	CodeNotFound         ResponseCode = 404 // 资源不存在
	CodeMethodNotAllowed ResponseCode = 405 // 方法不允许
	CodeRequestTimeout   ResponseCode = 408 // 请求超时
	CodeTooManyRequests  ResponseCode = 429 // 请求过于频繁

	// 服务端错误 (5xx)
	CodeInternalServerError ResponseCode = 500 // 服务器内部错误
	CodeNotImplemented      ResponseCode = 501 // 功能未实现
	CodeServiceUnavailable  ResponseCode = 503 // 服务不可用

	// 业务错误 (10xx)
	CodeValidationError ResponseCode = 1001 // 数据验证失败
	CodeDuplicateEntry  ResponseCode = 1002 // 重复条目
	CodePasswordError   ResponseCode = 1003 // 密码错误
	CodeUserDisabled    ResponseCode = 1004 // 用户已禁用
	CodeResourceInUse   ResponseCode = 1005 // 资源被占用
)

// Response 统一API响应结构
type Response struct {
	Code    ResponseCode  `json:"code"`             // 响应状态码
	Message string        `json:"message"`          // 响应消息
	Data    interface{}   `json:"data,omitempty"`   // 响应数据（可选）
	Meta    *Meta         `json:"meta,omitempty"`   // 元数据（分页等，可选）
	Errors  []ErrorDetail `json:"errors,omitempty"` // 错误详情列表（可选）
}

// Meta 元数据结构（用于分页等）
type Meta struct {
	Total      int64 `json:"total"`       // 总数
	Page       int   `json:"page"`        // 当前页
	PageSize   int   `json:"page_size"`   // 每页数量
	TotalPages int   `json:"total_pages"` // 总页数
}

// ErrorDetail 错误详情结构
type ErrorDetail struct {
	Field   string `json:"field,omitempty"` // 字段名（验证错误时使用）
	Message string `json:"message"`         // 错误消息
}

// AppError 应用错误结构
type AppError struct {
	Code    ResponseCode  // 错误码
	Message string        // 用户友好的错误消息
	Err     error         // 原始错误（用于调试）
	Details []ErrorDetail // 错误详情
}

// NewAppError 创建新的应用错误
func NewAppError(code ResponseCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// AddDetail 添加错误详情
func (e *AppError) AddDetail(field, message string) *AppError {
	e.Details = append(e.Details, ErrorDetail{
		Field:   field,
		Message: message,
	})
	return e
}

// WithDetails 设置错误详情
func (e *AppError) WithDetails(details []ErrorDetail) *AppError {
	e.Details = details
	return e
}

// getHTTPStatus 获取对应的HTTP状态码
func getHTTPStatus(code ResponseCode) int {
	switch {
	case code == CodeSuccess:
		return http.StatusOK
	case code >= 400 && code < 500:
		return int(code)
	case code >= 500 && code < 600:
		return int(code)
	case code >= 1000:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Success 成功响应（无数据）
func Success(c *gin.Context, message string) {
	localized := localizeMessage(c, message)
	c.Set(ContextResponseCodeKey, CodeSuccess)
	c.Set(ContextResponseMessageKey, localized)
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: localized,
	})
}

// SuccessData 成功响应（带数据）
func SuccessData(c *gin.Context, message string, data interface{}) {
	localized := localizeMessage(c, message)
	c.Set(ContextResponseCodeKey, CodeSuccess)
	c.Set(ContextResponseMessageKey, localized)
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: localized,
		Data:    data,
	})
}

// SuccessPaginated 成功响应（带分页数据）
func SuccessPaginated(c *gin.Context, message string, data interface{}, total int64, page, pageSize int) {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	localized := localizeMessage(c, message)
	c.Set(ContextResponseCodeKey, CodeSuccess)
	c.Set(ContextResponseMessageKey, localized)
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: localized,
		Data:    data,
		Meta: &Meta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// Created 创建成功响应
func Created(c *gin.Context, message string, data interface{}) {
	localized := localizeMessage(c, message)
	c.Set(ContextResponseCodeKey, CodeSuccess)
	c.Set(ContextResponseMessageKey, localized)
	c.JSON(http.StatusCreated, Response{
		Code:    CodeSuccess,
		Message: localized,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code ResponseCode, message string) {
	httpStatus := getHTTPStatus(code)
	localized := localizeMessage(c, message)
	c.Set(ContextResponseCodeKey, code)
	c.Set(ContextResponseMessageKey, localized)
	if _, exists := c.Get(ContextErrorCategoryKey); !exists {
		c.Set(ContextErrorCategoryKey, inferErrorCategory(code))
	}
	if _, exists := c.Get(ContextErrorReasonKey); !exists {
		c.Set(ContextErrorReasonKey, message)
	}
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: localized,
	})
}

// ErrorApp 应用错误响应
func ErrorApp(c *gin.Context, err *AppError) {
	httpStatus := getHTTPStatus(err.Code)
	localized := localizeMessage(c, err.Message)
	c.Set(ContextResponseCodeKey, err.Code)
	c.Set(ContextResponseMessageKey, localized)
	if _, exists := c.Get(ContextErrorCategoryKey); !exists {
		c.Set(ContextErrorCategoryKey, inferErrorCategory(err.Code))
	}
	if _, exists := c.Get(ContextErrorReasonKey); !exists {
		c.Set(ContextErrorReasonKey, err.Message)
	}
	resp := Response{
		Code:    err.Code,
		Message: localized,
	}
	if len(err.Details) > 0 {
		resp.Errors = err.Details
	}
	c.JSON(httpStatus, resp)
}

// ErrorValidation 验证错误响应
func ErrorValidation(c *gin.Context, message string, details []ErrorDetail) {
	localized := localizeMessage(c, message)
	c.Set(ContextResponseCodeKey, CodeValidationError)
	c.Set(ContextResponseMessageKey, localized)
	if _, exists := c.Get(ContextErrorCategoryKey); !exists {
		c.Set(ContextErrorCategoryKey, inferErrorCategory(CodeValidationError))
	}
	if _, exists := c.Get(ContextErrorReasonKey); !exists {
		c.Set(ContextErrorReasonKey, message)
	}
	c.JSON(http.StatusBadRequest, Response{
		Code:    CodeValidationError,
		Message: localized,
		Errors:  details,
	})
}

func inferErrorCategory(code ResponseCode) string {
	switch {
	case code == CodeSuccess:
		return ""
	case code == CodeUnauthorized:
		return "auth"
	case code == CodeForbidden:
		return "permission"
	case code == CodeTooManyRequests:
		return "rate_limit"
	case code == CodeValidationError || code == CodeBadRequest:
		return "validation"
	case code == CodeNotFound:
		return "not_found"
	case code >= 500 && code < 600:
		return "server"
	case code >= 1000:
		return "business"
	default:
		return "unknown"
	}
}

// SetErrorReason 设置标准化错误原因（建议使用 Reason* 常量）。
func SetErrorReason(c *gin.Context, reason string) {
	if c == nil {
		return
	}
	c.Set(ContextErrorReasonKey, reason)
}

// SetErrorCategory 设置标准化错误分类。
func SetErrorCategory(c *gin.Context, category string) {
	if c == nil {
		return
	}
	c.Set(ContextErrorCategoryKey, category)
}

// BadRequest 400错误响应
func BadRequest(c *gin.Context, message string) {
	Error(c, CodeBadRequest, message)
}

// Unauthorized 401错误响应
func Unauthorized(c *gin.Context, message string) {
	Error(c, CodeUnauthorized, message)
}

// Forbidden 403错误响应
func Forbidden(c *gin.Context, message string) {
	Error(c, CodeForbidden, message)
}

// NotFound 404错误响应
func NotFound(c *gin.Context, message string) {
	Error(c, CodeNotFound, message)
}

// InternalServerError 500错误响应
func InternalServerError(c *gin.Context, message string) {
	Error(c, CodeInternalServerError, message)
}

// TooManyRequests 429错误响应
func TooManyRequests(c *gin.Context, message string) {
	Error(c, CodeTooManyRequests, message)
}
