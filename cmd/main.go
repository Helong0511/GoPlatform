package main

import (
	"context"
	"fmt"
	"log"
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
	log.Println("Starting AD Platform Main Service...")

	// 加载配置
	log.Println("Loading configuration...")
	config, err := util.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded successfully")

	// 设置默认值
	log.Println("Setting defaults...")
	config.SetDefaults()
	log.Println("Defaults set")

	// 验证配置
	log.Println("Validating configuration...")
	if err := config.ValidateConfig(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	log.Println("Configuration validated successfully")

	// 启动监控服务器
	log.Printf("Starting metrics server on port %d...", config.Metrics.Port)
	metrics.StartServer(config.Metrics.Port, config.Metrics.Path)
	log.Println("Metrics server started")

	// 初始化数据库
	log.Println("Initializing database...")
	dbInstance, err := db.NewPostgresDB(&config.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database initialized successfully")

	// 初始化 Redis
	log.Println("Initializing Redis...")
	redisClient, err := redis.NewRedisClient(&config.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to redis: %v, running without cache", err)
		// 创建 mock 客户端，所有操作将不执行
		redisClient = redis.NewMockRedisClient()
	}
	log.Println("Redis initialized")

	// 初始化 Pulsar
	log.Println("Initializing Pulsar...")
	var pulsarProducer *pulsar.PulsarProducer = nil

	log.Printf("Info: Skipping Pulsar initialization for development")
	log.Println("Pulsar initialization skipped")

	// 初始化安全管理器
	log.Println("Initializing security manager...")
	securityManager, err := security.NewSecurityManager(config.Security.RustLibPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize security manager: %v", err)
		// 使用默认的安全管理器
		securityManager = &security.SecurityManager{}
	}
	log.Println("Security manager initialized")

	// 初始化服务
	log.Println("Initializing services...")
	userService := admin.NewUserService(dbInstance, redisClient)
	roleService := admin.NewRoleService(dbInstance)
	departmentService := admin.NewDepartmentService(dbInstance)
	adService := ad.NewAdService(dbInstance, redisClient)
	reportService := report.NewReportService(dbInstance, pulsarProducer)
	log.Println("Services initialized")

	// 启动 Pulsar 消费者
	// 由于在开发模式下跳过了 Pulsar 初始化，这里直接跳过消费者启动
	log.Printf("Info: Skipping Pulsar consumer startup")

	// 初始化速率限制器
	log.Println("Initializing rate limiter...")
	rateLimiter := middleware.NewRateLimiter(
		rate.Limit(config.RateLimit.RequestsPerSecond),
		config.RateLimit.Burst,
	)
	log.Println("Rate limiter initialized")

	aesSecret, aesSource, err := util.ResolveSecret(util.EnvAESSecretKey, config.Security.AESSecret)
	if err != nil {
		log.Fatalf("Failed to resolve AES secret: %v", err)
	}
	log.Printf("AES secret loaded from %s", aesSource)

	jwtSecret, jwtSource, err := util.ResolveSecret(util.EnvJWTSecret, config.JWT.Secret)
	if err != nil {
		log.Fatalf("Failed to resolve JWT secret: %v", err)
	}
	log.Printf("JWT secret loaded from %s", jwtSource)

	jwtExpiresIn, err := time.ParseDuration(config.JWT.ExpiresIn)
	if err != nil {
		log.Fatalf("Invalid JWT expires_in: %v", err)
	}

	var emailSender notify.EmailSender
	if config.Email.Enabled {
		emailSender = notify.NewSMTPEmailSender(config.Email)
		log.Printf("Email sender enabled: %s:%d", config.Email.Host, config.Email.Port)
	} else {
		log.Printf("Email sender disabled")
	}

	serverTimeout, err := time.ParseDuration(config.Server.Timeout)
	if err != nil {
		log.Printf("Warning: invalid server timeout, fallback to 30s: %v", err)
		serverTimeout = 30 * time.Second
	}

	// 初始化路由
	log.Println("Initializing router...")
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
		log.Fatalf("Failed to initialize router: %v", err)
	}
	log.Println("Router initialized")

	// 创建 Gin 引擎
	log.Println("Creating Gin engine...")
	engine := gin.Default()
	log.Println("Gin engine created")

	// 配置 CORS
	log.Println("Configuring CORS...")
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	log.Println("CORS configured")

	// 启用速率限制
	log.Println("Enabling rate limit...")
	engine.Use(rateLimiter.RateLimitMiddleware())
	log.Println("Rate limit enabled")

	// 启用多语言解析（默认简体中文）
	log.Println("Enabling locale middleware...")
	engine.Use(middleware.LocaleMiddleware())
	log.Println("Locale middleware enabled")

	// 设置路由
	log.Println("Setting up routes...")
	router.SetupRoutes(engine)
	log.Println("Routes setup completed")

	// 创建 HTTP 服务器
	log.Println("Creating HTTP server...")
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:           engine,
		ReadTimeout:       serverTimeout,
		WriteTimeout:      serverTimeout,
		IdleTimeout:       serverTimeout,
		ReadHeaderTimeout: serverTimeout,
	}
	log.Printf("HTTP server created on %s:%d", config.Server.Host, config.Server.Port)

	// 启动服务器
	log.Printf("Starting HTTP server on %s:%d...", config.Server.Host, config.Server.Port)
	go func() {
		log.Printf("Server is running on %s:%d", config.Server.Host, config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待服务器启动
	log.Printf("Server starting...")
	time.Sleep(2 * time.Second)
	log.Println("Server startup completed")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// 关闭服务
	reportService.Close()
	// 由于在开发模式下跳过了 Pulsar 初始化，这里直接跳过关闭操作
	redisClient.Close()
	dbInstance.Close()
	securityManager.Close()

	log.Println("Server exited")
}
