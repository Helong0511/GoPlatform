package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodyBytes 限制 POST/PUT/PATCH/DELETE 的请求体大小（读取至上限+1 字节判断超限）。
// maxBytes<=0 时不限制。超大包返回 413，避免下游 handler 忽略 Read 错误而覆盖状态码。
func MaxRequestBodyBytes(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			c.Next()
			return
		}
		if c.Request.Body == nil {
			c.Next()
			return
		}
		limited := io.LimitReader(c.Request.Body, maxBytes+1)
		data, err := io.ReadAll(limited)
		_ = c.Request.Body.Close()
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if int64(len(data)) > maxBytes {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(data))
		c.Request.ContentLength = int64(len(data))
		c.Next()
	}
}
