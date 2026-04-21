package system

import (
	"net/http"
	"time"

	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统相关处理器
type SystemHandler struct {
	// 可以添加需要的服务依赖
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// SetupRoutes 设置系统相关路由
func (h *SystemHandler) SetupRoutes(engine *gin.Engine) {
	// 根路径
	engine.GET("/", h.rootHandler)

	// 处理favicon.ico请求
	engine.GET("/favicon.ico", h.faviconHandler)

	// 健康检查
	engine.GET("/health", h.healthCheck)
}

// rootHandler 根路径处理
func (h *SystemHandler) rootHandler(c *gin.Context) {
	resp := gin.H{}
	resp["message"] = response.MsgWelcomeAdPlatformAPI
	resp["status"] = response.MsgStatusOK
	resp["version"] = "1.0.0"
	resp["endpoints"] = gin.H{
		"health":   "/health",
		"login":    "/api/login",
		"register": "/api/register",
		"ads":      "/api/ads",
		"report":   "/api/report",
		"admin":    "/api/admin",
	}
	c.JSON(http.StatusOK, resp)
}

// faviconHandler favicon处理
func (h *SystemHandler) faviconHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// healthCheck 健康检查
func (h *SystemHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  response.MsgStatusOK,
		"message": response.MsgServiceRunning,
		"time":    time.Now().UTC().Format(time.RFC3339), // 使用 UTC 标准时间
	})
}
