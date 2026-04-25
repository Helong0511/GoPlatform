package system

import (
	"net/http"
	"time"

	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统相关处理器
type SystemHandler struct {
	allowPublicRegister bool
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler(allowPublicRegister bool) *SystemHandler {
	return &SystemHandler{
		allowPublicRegister: allowPublicRegister,
	}
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

// SetupAdminRoutes 设置管理员系统路由
func (h *SystemHandler) SetupAdminRoutes(protected *gin.RouterGroup) {
	system := protected.Group("/system")
	{
		system.GET("/i18n-missing", middleware.RBACMiddleware("user:read"), h.getI18NMissing)
		system.POST("/i18n-missing/reset", middleware.RBACMiddleware("user:update"), h.resetI18NMissing)
	}
}

// rootHandler 根路径处理
func (h *SystemHandler) rootHandler(c *gin.Context) {
	resp := gin.H{}
	resp["message"] = response.Translate(c, response.MsgWelcomeAdPlatformAPI)
	resp["status"] = response.Translate(c, response.MsgStatusOK)
	resp["version"] = "1.0.0"
	endpoints := gin.H{
		"health": "/health",
		"login":  "/api/login",
		"ads":    "/api/ads",
		"report": "/api/report",
		"admin":  "/api/admin",
	}
	if h.allowPublicRegister {
		endpoints["register"] = "/api/register"
	}
	resp["endpoints"] = endpoints
	c.JSON(http.StatusOK, resp)
}

// faviconHandler favicon处理
func (h *SystemHandler) faviconHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// healthCheck 健康检查
func (h *SystemHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  response.Translate(c, response.MsgStatusOK),
		"message": response.Translate(c, response.MsgServiceRunning),
		"time":    time.Now().UTC().Format(time.RFC3339), // 使用 UTC 标准时间
	})
}

func (h *SystemHandler) getI18NMissing(c *gin.Context) {
	response.SuccessData(c, response.MsgSuccess, gin.H{
		"missing": response.ListMissingTranslations(),
	})
}

func (h *SystemHandler) resetI18NMissing(c *gin.Context) {
	response.ResetMissingTranslations()
	response.Success(c, response.MsgSuccess)
}
