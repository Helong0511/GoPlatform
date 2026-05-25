package system

import (
	"context"
	"net/http"
	"time"

	"ad-platform/pkg/logger"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统相关处理器
type SystemHandler struct {
	allowPublicRegister bool
	readiness           func(context.Context) error
	ipGuard             *middleware.IPGuard
	configStore         *util.ConfigStore
}

// NewSystemHandler 创建系统处理器；readiness 非 nil 时注册 /health/ready（DB/Redis 等依赖探测）。
func NewSystemHandler(allowPublicRegister bool, readiness func(context.Context) error, ipGuard *middleware.IPGuard, configStore *util.ConfigStore) *SystemHandler {
	return &SystemHandler{
		allowPublicRegister: allowPublicRegister,
		readiness:           readiness,
		ipGuard:             ipGuard,
		configStore:         configStore,
	}
}

// SetupRoutes 设置系统相关路由
func (h *SystemHandler) SetupRoutes(engine *gin.Engine) {
	// 根路径
	engine.GET("/", h.rootHandler)

	// 处理favicon.ico请求
	engine.GET("/favicon.ico", h.faviconHandler)

	// 存活探针（轻量，不访问外部依赖）
	engine.GET("/health", h.healthLiveness)
	engine.GET("/health/live", h.healthLiveness)
	if h.readiness != nil {
		engine.GET("/health/ready", h.healthReadiness)
	}
}

// SetupAdminRoutes 设置管理员系统路由
func (h *SystemHandler) SetupAdminRoutes(protected *gin.RouterGroup) {
	system := protected.Group("/system")
	{
		system.GET("/i18n-missing", middleware.RBACMiddleware("user:read"), h.getI18NMissing)
		system.POST("/i18n-missing/reset", middleware.RBACMiddleware("user:update"), h.resetI18NMissing)
		if h.configStore != nil {
			system.GET("/config", middleware.RBACMiddleware("user:read"), h.getSystemConfig)
			system.PUT("/config", middleware.RBACMiddleware("user:update"), h.putSystemConfig)
			system.POST("/config/apply-hot", middleware.RBACMiddleware("user:update"), h.applyHotSystemConfig)
		}
		if h.ipGuard != nil {
			system.GET("/ip-access/status", middleware.RBACMiddleware("user:read"), h.getIPAccessStatus)
			system.GET("/ip-blacklist", middleware.RBACMiddleware("user:read"), h.listIPBlacklist)
			system.POST("/ip-blacklist", middleware.RBACMiddleware("user:update"), h.banIP)
			system.DELETE("/ip-blacklist/*ip", middleware.RBACMiddleware("user:update"), h.unbanIP)
			system.GET("/ip-whitelist", middleware.RBACMiddleware("user:read"), h.listIPWhitelist)
			system.POST("/ip-whitelist", middleware.RBACMiddleware("user:update"), h.addIPWhitelist)
			system.DELETE("/ip-whitelist/*ip", middleware.RBACMiddleware("user:update"), h.removeIPWhitelist)
		}
	}
}

// rootHandler 根路径处理
func (h *SystemHandler) rootHandler(c *gin.Context) {
	resp := gin.H{}
	resp["message"] = response.Translate(c, response.MsgWelcomeAdPlatformAPI)
	resp["status"] = response.Translate(c, response.MsgStatusOK)
	resp["version"] = util.AppVersion()
	endpoints := gin.H{
		"health":       "/health",
		"health_live":  "/health/live",
		"login":        "/api/login",
		"report":       "/api/report",
		"admin":        "/api/admin",
		"admin_ads":    "/api/admin/ads",
		"admin_reports": "/api/admin/reports",
	}
	if h.allowPublicRegister {
		endpoints["register"] = "/api/register"
	}
	if h.readiness != nil {
		endpoints["health_ready"] = "/health/ready"
	}
	resp["endpoints"] = endpoints
	c.JSON(http.StatusOK, resp)
}

// faviconHandler favicon处理
func (h *SystemHandler) faviconHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (h *SystemHandler) healthLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  response.Translate(c, response.MsgStatusOK),
		"message": response.Translate(c, response.MsgServiceRunning),
		"time":    time.Now().UTC().Format(time.RFC3339),
		"version": util.AppVersion(),
	})
}

func (h *SystemHandler) healthReadiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := h.readiness(ctx); err != nil {
		logger.S().Warn("readiness check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"time":    time.Now().UTC().Format(time.RFC3339),
			"version": util.AppVersion(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  response.Translate(c, response.MsgStatusOK),
		"ready":   true,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"version": util.AppVersion(),
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

