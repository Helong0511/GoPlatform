package api

import (
	"context"
	"time"

	"ad-platform/api/ad"
	"ad-platform/api/auth"
	"ad-platform/api/client"
	"ad-platform/api/department"
	"ad-platform/api/report"
	"ad-platform/api/role"
	"ad-platform/api/system"
	"ad-platform/api/user"
	adInternal "ad-platform/internal/ad"
	"ad-platform/internal/admin"
	reportInternal "ad-platform/internal/report"
	"ad-platform/internal/security"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/notify"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// Router 路由管理器
type Router struct {
	userService            *admin.UserService
	roleService            *admin.RoleService
	departmentService      *admin.DepartmentService
	adService              *adInternal.AdService
	reportService          *reportInternal.ReportService
	securityManager        *security.SecurityManager
	jwtSecret              string
	jwtIssuer              string
	jwtAudience            string
	jwtExpiresIn           time.Duration
	allowPublicRegister    bool
	emailSender            notify.EmailSender
	readinessCheck         func(context.Context) error
	loginRateMiddleware    gin.HandlerFunc
	registerRateMiddleware gin.HandlerFunc
	ipGuard                *middleware.IPGuard
	configStore            *util.ConfigStore
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
	jwtIssuer string,
	jwtAudience string,
	jwtExpiresIn time.Duration,
	allowPublicRegister bool,
	emailSender notify.EmailSender,
	readinessCheck func(context.Context) error,
	registerRateMiddleware gin.HandlerFunc,
	loginRateMiddleware gin.HandlerFunc,
	ipGuard *middleware.IPGuard,
	configStore *util.ConfigStore,
) *Router {
	return &Router{
		userService:            userService,
		roleService:            roleService,
		departmentService:      departmentService,
		adService:              adService,
		reportService:          reportService,
		securityManager:        securityManager,
		jwtSecret:              jwtSecret,
		jwtIssuer:              jwtIssuer,
		jwtAudience:            jwtAudience,
		jwtExpiresIn:           jwtExpiresIn,
		allowPublicRegister:    allowPublicRegister,
		emailSender:            emailSender,
		readinessCheck:         readinessCheck,
		registerRateMiddleware: registerRateMiddleware,
		loginRateMiddleware:    loginRateMiddleware,
		ipGuard:                ipGuard,
		configStore:            configStore,
	}
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes(engine *gin.Engine) {
	systemHandler := system.NewSystemHandler(r.allowPublicRegister, r.readinessCheck, r.ipGuard, r.configStore)
	authHandler := auth.NewAuthHandler(r.userService, r.jwtSecret, r.jwtIssuer, r.jwtAudience, r.jwtExpiresIn, r.allowPublicRegister, r.emailSender)
	userHandler := user.NewUserHandler(r.userService, r.emailSender)
	roleHandler := role.NewRoleHandler(r.roleService)
	departmentHandler := department.NewDepartmentHandler(r.departmentService)
	clientHandler := client.NewClientHandler(r.userService)
	adHandler := ad.NewAdHandler(r.adService)
	reportHandler := report.NewReportHandler(r.reportService)

	systemHandler.SetupRoutes(engine)

	public := engine.Group("/api")
	protected := engine.Group("/api/admin")

	authHandler.SetupRoutes(public, protected, r.loginRateMiddleware, r.registerRateMiddleware)
	protected.Use(middleware.UserContextMiddleware(r.userService))
	systemHandler.SetupAdminRoutes(protected)

	userHandler.SetupRoutes(protected)
	roleHandler.SetupRoutes(protected)
	departmentHandler.SetupRoutes(protected)
	clientHandler.SetupRoutes(protected)

	adHandler.SetupRoutes(protected)

	reportAPIKeyGuard := middleware.APIKeyMiddleware(r.userService, "report:write")
	reportHandler.SetupRoutes(public, protected, reportAPIKeyGuard)
}
