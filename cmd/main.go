package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ad-platform/api"
	"ad-platform/internal/ad"
	"ad-platform/internal/admin"
	"ad-platform/internal/report"
	"ad-platform/internal/security"
	"ad-platform/pkg/db"
	"ad-platform/pkg/logger"
	"ad-platform/pkg/metrics"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/notify"
	"ad-platform/pkg/pulsar"
	"ad-platform/pkg/redis"
	"ad-platform/pkg/util"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func main() {
	logger.Init("info", "json")
	logger.S().Info("starting AD Platform Main Service")

	// 加载配置
	configPath, err := resolveConfigPath()
	if err != nil {
		logger.S().Error("failed to resolve config path", "error", err)
		os.Exit(1)
	}
	logger.S().Info("loading configuration", "config_path", configPath)
	config, err := util.LoadConfig(configPath)
	if err != nil {
		logger.S().Error("failed to load config", "error", err)
		os.Exit(1)
	}
	logger.Init(config.Logging.Level, config.Logging.Format)
	logger.S().Info("configuration loaded successfully",
		"env", config.Environment,
		"log_level", config.Logging.Level,
		"log_format", config.Logging.Format,
	)

	// 设置默认值
	logger.S().Info("setting defaults")
	config.SetDefaults()
	logger.S().Info("defaults set")

	// 验证配置
	logger.S().Info("validating configuration")
	if err := config.ValidateConfig(); err != nil {
		logger.S().Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger.S().Info("configuration validated successfully")

	// 启动监控服务器
	logger.S().Info("starting metrics server", "port", config.Metrics.Port, "path", config.Metrics.Path)
	metrics.StartServer(config.Metrics.Port, config.Metrics.Path)
	logger.S().Info("metrics server started")

	// 初始化数据库
	logger.S().Info("initializing database")
	dbInstance, err := db.NewPostgresDB(&config.Database)
	if err != nil {
		logger.S().Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.S().Info("database initialized successfully")

	// 初始化 Redis
	logger.S().Info("initializing redis")
	redisClient, err := redis.NewRedisClient(&config.Redis)
	if err != nil {
		logger.S().Warn("failed to connect redis, running without cache", "error", err)
		// 创建 mock 客户端，所有操作将不执行
		redisClient = redis.NewMockRedisClient()
	}
	logger.S().Info("redis initialized")

	// 初始化 Pulsar
	logger.S().Info("initializing pulsar")
	var pulsarProducer *pulsar.PulsarProducer = nil

	logger.S().Info("skipping pulsar initialization for development")

	// 初始化安全管理器
	logger.S().Info("initializing security manager")
	securityManager, err := security.NewSecurityManager(config.Security.RustLibPath)
	if err != nil {
		logger.S().Warn("failed to initialize security manager, using fallback", "error", err)
		// 使用默认的安全管理器
		securityManager = &security.SecurityManager{}
	}
	logger.S().Info("security manager initialized")

	// 初始化服务
	logger.S().Info("initializing services")
	userService := admin.NewUserService(dbInstance, redisClient)
	roleService := admin.NewRoleService(dbInstance)
	departmentService := admin.NewDepartmentService(dbInstance)
	adService := ad.NewAdService(dbInstance, redisClient)
	reportService := report.NewReportService(dbInstance, pulsarProducer)
	logger.S().Info("services initialized")

	// 启动 Pulsar 消费者
	// 由于在开发模式下跳过了 Pulsar 初始化，这里直接跳过消费者启动
	logger.S().Info("skipping pulsar consumer startup")

	// 初始化速率限制器
	logger.S().Info("initializing rate limiter")
	rateLimiter := middleware.NewRateLimiter(
		rate.Limit(config.RateLimit.RequestsPerSecond),
		config.RateLimit.Burst,
	)
	logger.S().Info("rate limiter initialized",
		"requests_per_second", config.RateLimit.RequestsPerSecond,
		"burst", config.RateLimit.Burst,
	)

	aesSecret, aesSource, err := util.ResolveSecret(util.EnvAESSecretKey, config.Security.AESSecret)
	if err != nil {
		logger.S().Error("failed to resolve AES secret", "error", err)
		os.Exit(1)
	}
	logger.S().Info("AES secret loaded", "source", aesSource)

	jwtSecret, jwtSource, err := util.ResolveSecret(util.EnvJWTSecret, config.JWT.Secret)
	if err != nil {
		logger.S().Error("failed to resolve JWT secret", "error", err)
		os.Exit(1)
	}
	logger.S().Info("JWT secret loaded", "source", jwtSource)

	jwtExpiresIn, err := time.ParseDuration(config.JWT.ExpiresIn)
	if err != nil {
		logger.S().Error("invalid JWT expires_in", "error", err)
		os.Exit(1)
	}

	var emailSender notify.EmailSender
	if config.Email.Enabled {
		emailSender = notify.NewSMTPEmailSender(config.Email)
		logger.S().Info("email sender enabled", "host", config.Email.Host, "port", config.Email.Port)
	} else {
		logger.S().Info("email sender disabled")
	}

	serverTimeout, err := time.ParseDuration(config.Server.Timeout)
	if err != nil {
		logger.S().Warn("invalid server timeout, fallback to 30s", "error", err)
		serverTimeout = 30 * time.Second
	}

	// 初始化路由
	logger.S().Info("initializing router")
	router, err := api.NewRouter(
		userService,
		roleService,
		departmentService,
		adService,
		reportService,
		securityManager,
		jwtSecret,
		jwtExpiresIn,
		config.PublicRegisterEnabled(),
		aesSecret,
		emailSender,
	)
	if err != nil {
		logger.S().Error("failed to initialize router", "error", err)
		os.Exit(1)
	}
	logger.S().Info("router initialized")

	// 创建 Gin 引擎
	logger.S().Info("creating Gin engine")
	engine := gin.New()
	engine.Use(
		logger.RequestIDMiddleware(),
		logger.AccessLogMiddlewareWithOptions(logger.AccessLogOptions{
			SlowRequestThreshold: time.Duration(config.Logging.SlowRequestMS) * time.Millisecond,
		}),
		logger.RecoveryMiddleware(),
	)
	logger.S().Info("Gin engine created with custom logging middlewares")

	// 配置 CORS
	logger.S().Info("configuring CORS")
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	logger.S().Info("CORS configured")

	// 启用速率限制
	logger.S().Info("enabling rate limit")
	engine.Use(rateLimiter.RateLimitMiddleware())
	logger.S().Info("rate limit enabled")

	// 启用多语言解析（默认简体中文）
	logger.S().Info("enabling locale middleware")
	engine.Use(middleware.LocaleMiddleware())
	logger.S().Info("locale middleware enabled")

	// 设置路由
	logger.S().Info("setting up routes")
	router.SetupRoutes(engine)
	logger.S().Info("routes setup completed")

	// 创建 HTTP 服务器
	logger.S().Info("creating HTTP server")
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:           engine,
		ReadTimeout:       serverTimeout,
		WriteTimeout:      serverTimeout,
		IdleTimeout:       serverTimeout,
		ReadHeaderTimeout: serverTimeout,
	}
	logger.S().Info("HTTP server created", "host", config.Server.Host, "port", config.Server.Port)

	// 启动服务器
	logger.S().Info("starting HTTP server", "host", config.Server.Host, "port", config.Server.Port)
	go func() {
		logger.S().Info("server is running", "host", config.Server.Host, "port", config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.S().Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// 等待服务器启动
	logger.S().Info("server starting")
	time.Sleep(2 * time.Second)
	logger.S().Info("server startup completed")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.S().Info("shutting down server")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.S().Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// 关闭服务
	reportService.Close()
	// 由于在开发模式下跳过了 Pulsar 初始化，这里直接跳过关闭操作
	redisClient.Close()
	dbInstance.Close()
	securityManager.Close()

	logger.S().Info("server exited")
}

func resolveConfigPath() (string, error) {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	candidates := []string{
		"config/config.yaml",
		"config.yaml",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("no config file found, tried CONFIG_PATH, config/config.yaml, config.yaml")
}
