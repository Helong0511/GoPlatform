package api

import (
	"ad-platform/api/ad"
	"ad-platform/api/auth"
	"ad-platform/api/client"
	"ad-platform/api/department"
	"ad-platform/api/report"
	"ad-platform/api/role"
	"ad-platform/api/system"
	"ad-platform/api/user"
	"ad-platform/internal/admin"
	adInternal "ad-platform/internal/ad"
	reportInternal "ad-platform/internal/report"
	"ad-platform/internal/security"

	"github.com/gin-gonic/gin"
)

// Router 路由管理器
type Router struct {
	userService       *admin.UserService
	roleService       *admin.RoleService
	departmentService *admin.DepartmentService
	adService         *adInternal.AdService
	reportService     *reportInternal.ReportService
	securityManager   *security.SecurityManager
	jwtSecret         string
	aesUtils          *security.AESUtils
}

// NewRouter 创建路由管理器
func NewRouter(
	userService *admin.UserService,
	roleService *admin.RoleService,
	departmentService *admin.DepartmentService,
	adService *adInternal.AdService,
	reportService *reportInternal.ReportService,
	securityManager *security.SecurityManager,
	jwtSecret string,
	aesSecret string,
) (*Router, error) {
	// 初始化AES工具 - 使用配置的密钥
	aesUtils, err := security.NewAESUtils(aesSecret)
	if err != nil {
		return nil, err
	}

	return &Router{
		userService:       userService,
		roleService:       roleService,
		departmentService: departmentService,
		adService:         adService,
		reportService:     reportService,
		securityManager:   securityManager,
		jwtSecret:         jwtSecret,
		aesUtils:          aesUtils,
	}, nil
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// 初始化各个模块的处理器
	systemHandler := system.NewSystemHandler()
	authHandler := auth.NewAuthHandler(r.userService, r.jwtSecret)
	userHandler := user.NewUserHandler(r.userService)
	roleHandler := role.NewRoleHandler(r.roleService)
	departmentHandler := department.NewDepartmentHandler(r.departmentService)
	clientHandler := client.NewClientHandler(r.userService)
	adHandler := ad.NewAdHandler(r.adService, r.aesUtils)
	reportHandler := report.NewReportHandler(r.reportService)

	// 设置系统相关路由
	systemHandler.SetupRoutes(engine)

	// 公开接口
	public := engine.Group("/api")

	// 需要认证的接口
	protected := engine.Group("/api/admin")

	// 设置认证相关路由
	authHandler.SetupRoutes(public, protected)

	// 设置用户相关路由
	userHandler.SetupRoutes(protected)

	// 设置角色相关路由
	roleHandler.SetupRoutes(protected)

	// 设置部门相关路由
	departmentHandler.SetupRoutes(protected)

	// 设置客户相关路由
	clientHandler.SetupRoutes(protected)

	// 设置广告相关路由
	adHandler.SetupRoutes(engine, public, protected)

	// 设置报告相关路由
	reportHandler.SetupRoutes(engine, public, protected)
}
