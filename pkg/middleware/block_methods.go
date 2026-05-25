package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// allowedHTTPMethods API 允许的方法白名单（含 OPTIONS 以配合 CORS 预检）。
var allowedHTTPMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// BlockUnsafeHTTPMethods 拒绝非白名单 HTTP 方法（含 TRACE/TRACK/CONNECT 等）。
func BlockUnsafeHTTPMethods() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
		if _, ok := allowedHTTPMethods[method]; !ok {
			c.AbortWithStatus(http.StatusMethodNotAllowed)
			return
		}
		c.Next()
	}
}
