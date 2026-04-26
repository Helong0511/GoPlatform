package logger

import (
	"ad-platform/pkg/response"
	"ad-platform/pkg/telemetry"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader 请求ID响应头。
	RequestIDHeader = "X-Request-ID"
	// RequestIDContextKey gin上下文中的请求ID键。
	RequestIDContextKey = "request_id"
)

// AccessLogOptions 访问日志配置项。
type AccessLogOptions struct {
	SlowRequestThreshold time.Duration
}

func normalizeAccessLogOptions(opts AccessLogOptions) AccessLogOptions {
	if opts.SlowRequestThreshold <= 0 {
		opts.SlowRequestThreshold = 300 * time.Millisecond
	}
	return opts
}

// RequestIDMiddleware 生成/透传请求ID并写入上下文。
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if reqID == "" {
			reqID = newRequestID()
		}

		c.Set(RequestIDContextKey, reqID)
		c.Writer.Header().Set(RequestIDHeader, reqID)
		c.Next()
	}
}

// AccessLogMiddleware 记录超详细HTTP访问日志。
func AccessLogMiddleware() gin.HandlerFunc {
	return AccessLogMiddlewareWithOptions(AccessLogOptions{})
}

// AccessLogMiddlewareWithOptions 记录超详细HTTP访问日志（可配置）。
func AccessLogMiddlewareWithOptions(opts AccessLogOptions) gin.HandlerFunc {
	opts = normalizeAccessLogOptions(opts)
	httpLogger := Component("http")

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := sanitizeQuery(c.Request.URL.RawQuery)
		method := c.Request.Method
		clientIP := c.ClientIP()
		ua := c.Request.UserAgent()
		reqID := RequestIDFromContext(c)

		// 启动日志只在debug级别输出，避免生产噪音。
		httpLogger.Debug("http request started",
			"request_id", reqID,
			"method", method,
			"path", path,
			"query", query,
			"client_ip", clientIP,
			"user_agent", ua,
			"authorization", maskCredential(c.GetHeader("Authorization")),
			"x_api_key", maskCredential(c.GetHeader("X-API-Key")),
		)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		entry := []any{
			"request_id", reqID,
			"method", method,
			"path", path,
			"query", query,
			"route", c.FullPath(),
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"latency_human", latency.String(),
			"client_ip", clientIP,
			"user_agent", ua,
			"request_size", c.Request.ContentLength,
			"response_size", c.Writer.Size(),
			"error_count", len(c.Errors),
		}
		if responseCode, exists := c.Get(response.ContextResponseCodeKey); exists {
			entry = append(entry, "error_code", fmt.Sprintf("%v", responseCode))
		}
		if responseMessage, exists := c.Get(response.ContextResponseMessageKey); exists {
			entry = append(entry, "response_message", fmt.Sprintf("%v", responseMessage))
		}
		if errorCategory, exists := c.Get(response.ContextErrorCategoryKey); exists {
			entry = append(entry, "error_category", fmt.Sprintf("%v", errorCategory))
		}
		if errorReason, exists := c.Get(response.ContextErrorReasonKey); exists {
			entry = append(entry, "error_reason", fmt.Sprintf("%v", errorReason))
		}

		if status >= http.StatusBadRequest {
			telemetry.RecordHTTPError(
				stringValueFromContext(c, response.ContextResponseCodeKey),
				stringValueFromContext(c, response.ContextErrorCategoryKey),
				stringValueFromContext(c, response.ContextErrorReasonKey),
			)
		}

		if apiUserID, exists := c.Get("api_user_id"); exists {
			entry = append(entry, "api_user_id", fmt.Sprintf("%v", apiUserID))
		}

		if len(c.Errors) > 0 {
			entry = append(entry, "errors", c.Errors.String())
		}

		switch {
		case status >= http.StatusInternalServerError:
			httpLogger.Error("http request completed", entry...)
		case latency >= opts.SlowRequestThreshold:
			entry = append(entry, "slow_request_threshold_ms", opts.SlowRequestThreshold.Milliseconds())
			httpLogger.Warn("http slow request", entry...)
		case status >= http.StatusBadRequest:
			httpLogger.Warn("http request completed", entry...)
		default:
			httpLogger.Info("http request completed", entry...)
		}
	}
}

// RecoveryMiddleware 捕获panic并输出完整堆栈日志。
func RecoveryMiddleware() gin.HandlerFunc {
	panicLogger := Component("panic")

	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		panicLogger.Error("panic recovered",
			"request_id", RequestIDFromContext(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"client_ip", c.ClientIP(),
			"panic", fmt.Sprintf("%v", recovered),
			"stack", string(debug.Stack()),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
		})
	})
}

// RequestIDFromContext 从gin上下文中读取请求ID。
func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	v, exists := c.Get(RequestIDContextKey)
	if !exists {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func newRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func maskCredential(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return "****"
	}
	return v[:4] + "****" + v[len(v)-2:]
}

func sanitizeQuery(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	vals, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}

	for key, value := range vals {
		k := strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(k, "token") || strings.Contains(k, "secret") || strings.Contains(k, "password") || strings.Contains(k, "apikey") || strings.Contains(k, "api_key") {
			if len(value) == 0 {
				vals.Set(key, "****")
				continue
			}
			for i := range value {
				value[i] = "****"
			}
			vals[key] = value
		}
	}

	return vals.Encode()
}

func stringValueFromContext(c *gin.Context, key string) string {
	if c == nil {
		return ""
	}
	v, exists := c.Get(key)
	if !exists {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
