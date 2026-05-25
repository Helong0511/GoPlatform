package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 为 API 响应添加基础安全头。
// hstsMaxAgeSeconds > 0 时追加 Strict-Transport-Security（仅在全站 HTTPS、TLS 终端配置正确时启用）。
// trustForwardedProto 为 true 时才根据 X-Forwarded-Proto 判定 HTTPS（需已配置 gin trusted_proxies）。
func SecurityHeadersMiddleware(hstsMaxAgeSeconds int, trustForwardedProto bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if hstsMaxAgeSeconds > 0 && ClientScheme(c, trustForwardedProto) == "https" {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", hstsMaxAgeSeconds))
		}
		c.Next()
	}
}

// ClientScheme 返回客户端认为的 scheme（小写 http/https）。
func ClientScheme(c *gin.Context, trustForwardedProto bool) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if trustForwardedProto {
		if p := strings.TrimSpace(strings.ToLower(c.GetHeader("X-Forwarded-Proto"))); p != "" {
			return p
		}
	}
	if c.Request.TLS != nil {
		return "https"
	}
	if c.Request.URL != nil {
		if sch := strings.ToLower(strings.TrimSpace(c.Request.URL.Scheme)); sch == "https" {
			return "https"
		}
	}
	return "http"
}
