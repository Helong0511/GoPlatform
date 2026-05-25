package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	dotenvPath, dotenvErr := util.LoadDotEnvFiles()
	if dotenvErr != nil {
		logger.S().Warn("failed to load .env file", "error", dotenvErr)
	} else if dotenvPath != "" {
		logger.S().Info("loaded environment from dotenv file", "path", dotenvPath)
	}

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
	logger.S().Info("starting metrics server",
		"host", config.Metrics.Host,
		"port", config.Metrics.Port,
		"path", config.Metrics.Path,
		"bearer_auth", strings.TrimSpace(config.Metrics.BearerToken) != "",
	)
	metricsSrv := metrics.StartServer(config.Metrics.Host, config.Metrics.Port, config.Metrics.Path, config.Metrics.BearerToken)
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

	// 初始化 Pulsar（url 为空则跳过，便于无消息队列环境）
	var pulsarProducer *pulsar.PulsarProducer
	var pulsarConsumer *pulsar.PulsarConsumer
	if pulsarURL := strings.TrimSpace(config.Pulsar.URL); pulsarURL != "" {
		logger.S().Info("initializing pulsar", "url", pulsarURL)
		pulsarProducer, err = pulsar.NewPulsarProducer(pulsarURL, config.Pulsar.Topic)
		if err != nil {
			if util.IsProductionEnvironment(config.Environment) {
				logger.S().Error("failed to initialize pulsar producer in production", "error", err)
				os.Exit(1)
			}
			logger.S().Warn("failed to initialize pulsar producer, running without queue", "error", err)
			pulsarProducer = nil
		} else {
			pulsarConsumer, err = pulsar.NewPulsarConsumer(pulsarURL, config.Pulsar.Topic, config.Pulsar.Subscription)
			if err != nil {
				pulsarProducer.Close()
				pulsarProducer = nil
				if util.IsProductionEnvironment(config.Environment) {
					logger.S().Error("failed to initialize pulsar consumer in production", "error", err)
					os.Exit(1)
				}
				logger.S().Warn("failed to initialize pulsar consumer, running without queue", "error", err)
			} else {
				logger.S().Info("pulsar producer and consumer initialized")
			}
		}
	} else {
		logger.S().Info("pulsar url empty, skipping message queue")
	}

	// 初始化安全管理器
	logger.S().Info("initializing security manager")
	securityManager, err := security.NewSecurityManager(config.Security.RustLibPath)
	if err != nil {
		logger.S().Error("failed to initialize security manager", "error", err)
		os.Exit(1)
	}
	logger.S().Info("security manager initialized")

	// 初始化服务
	logger.S().Info("initializing services")
	userService := admin.NewUserService(dbInstance, redisClient, config.Auth.MinPasswordLength)
	roleService := admin.NewRoleService(dbInstance)
	departmentService := admin.NewDepartmentService(dbInstance)
	adService := ad.NewAdService(dbInstance, redisClient)
	reportService := report.NewReportService(dbInstance, pulsarProducer)
	logger.S().Info("services initialized")

	if pulsarConsumer != nil {
		go reportService.ConsumeReports(pulsarConsumer)
		logger.S().Info("pulsar consumer started")
	}

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

	var loginRateMiddleware gin.HandlerFunc
	if config.RateLimit.LoginRequestsPerSecond > 0 {
		loginRL := middleware.NewRateLimiter(
			rate.Limit(config.RateLimit.LoginRequestsPerSecond),
			config.RateLimit.LoginBurst,
		)
		loginRateMiddleware = loginRL.RateLimitMiddleware()
		logger.S().Info("login rate limiter enabled",
			"login_requests_per_second", config.RateLimit.LoginRequestsPerSecond,
			"login_burst", config.RateLimit.LoginBurst,
		)
	}

	var registerRateMiddleware gin.HandlerFunc
	if config.RateLimit.RegisterRequestsPerSecond > 0 {
		regRL := middleware.NewRateLimiter(
			rate.Limit(config.RateLimit.RegisterRequestsPerSecond),
			config.RateLimit.RegisterBurst,
		)
		registerRateMiddleware = regRL.RateLimitMiddleware()
		logger.S().Info("register rate limiter enabled",
			"register_requests_per_second", config.RateLimit.RegisterRequestsPerSecond,
			"register_burst", config.RateLimit.RegisterBurst,
		)
	}

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

	if util.IsStrictHardeningEnvironment(config.Environment) {
		if err := util.ValidateProductionSecrets(jwtSecret, aesSecret); err != nil {
			logger.S().Error("strict environment secret validation failed", "error", err, "env", config.Environment)
			os.Exit(1)
		}
		logger.S().Info("strict environment secret strength validated", "env", config.Environment)
	}

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

	shutdownTimeout, err := time.ParseDuration(config.Server.ShutdownTimeout)
	if err != nil {
		logger.S().Warn("invalid shutdown_timeout, fallback to 15s", "error", err)
		shutdownTimeout = 15 * time.Second
	}
	maxBodyBytes := int64(config.Server.MaxRequestBodyMB) * 1024 * 1024

	readinessCheck := func(ctx context.Context) error {
		sqlDB, err := dbInstance.DB.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			return err
		}
		if redisClient.Client != nil {
			if err := redisClient.Client.Ping(ctx).Err(); err != nil {
				return err
			}
		}
		return nil
	}

	ipGuardOpts, err := middleware.ParseIPGuardOptions(middleware.IPBlacklistYAML{
		Enabled:               config.Security.IPBlacklist.Enabled,
		BanDuration:           config.Security.IPBlacklist.BanDuration,
		StrikeWindow:          config.Security.IPBlacklist.StrikeWindow,
		StrikesBeforeBan:      config.Security.IPBlacklist.StrikesBeforeBan,
		BlockCrawlerUserAgent: config.Security.IPBlacklist.BlockCrawlerUserAgent,
		CrawlerStrikeWeight:   config.Security.IPBlacklist.CrawlerStrikeWeight,
		StaticIPs:             config.Security.IPBlacklist.StaticIPs,
		StaticWhitelistIPs:    config.Security.IPBlacklist.StaticWhitelistIPs,
	})
	if err != nil {
		logger.S().Error("invalid ip blacklist configuration", "error", err)
		os.Exit(1)
	}
	ipGuard := middleware.NewIPGuard(ipGuardOpts, redisClient)

	configStore := util.NewConfigStore(configPath, dotenvPath)
	configStore.RegisterHotReload("security.ip_blacklist", func(cfg *util.Config) error {
		opts, err := middleware.ParseIPGuardOptions(middleware.IPBlacklistYAML{
			Enabled:               cfg.Security.IPBlacklist.Enabled,
			BanDuration:           cfg.Security.IPBlacklist.BanDuration,
			StrikeWindow:          cfg.Security.IPBlacklist.StrikeWindow,
			StrikesBeforeBan:      cfg.Security.IPBlacklist.StrikesBeforeBan,
			BlockCrawlerUserAgent: cfg.Security.IPBlacklist.BlockCrawlerUserAgent,
			CrawlerStrikeWeight:   cfg.Security.IPBlacklist.CrawlerStrikeWeight,
			StaticIPs:             cfg.Security.IPBlacklist.StaticIPs,
			StaticWhitelistIPs:    cfg.Security.IPBlacklist.StaticWhitelistIPs,
		})
		if err != nil {
			return err
		}
		ipGuard.ApplyOptions(opts)
		return nil
	})

	if ipGuard.Enabled() {
		logger.S().Info("ip blacklist guard enabled",
			"ban_duration", ipGuardOpts.BanDuration.String(),
			"strike_window", ipGuardOpts.StrikeWindow.String(),
			"strikes_before_ban", ipGuardOpts.StrikesBeforeBan,
		)
	} else {
		logger.S().Info("ip blacklist guard disabled")
	}

	// 初始化路由
	logger.S().Info("initializing router")
	router := api.NewRouter(
		userService,
		roleService,
		departmentService,
		adService,
		reportService,
		securityManager,
		jwtSecret,
		config.JWT.Issuer,
		config.JWT.Audience,
		jwtExpiresIn,
		config.PublicRegisterEnabled(),
		emailSender,
		readinessCheck,
		registerRateMiddleware,
		loginRateMiddleware,
		ipGuard,
		configStore,
	)
	logger.S().Info("router initialized")

	if util.IsProductionEnvironment(config.Environment) {
		gin.SetMode(gin.ReleaseMode)
		logger.S().Info("gin mode set to release (production)")
	}

	// 创建 Gin 引擎
	logger.S().Info("creating Gin engine")
	engine := gin.New()
	if len(config.Server.TrustedProxies) > 0 {
		if err := engine.SetTrustedProxies(config.Server.TrustedProxies); err != nil {
			logger.S().Error("invalid server.trusted_proxies", "error", err)
			os.Exit(1)
		}
		logger.S().Info("gin trusted proxies configured", "count", len(config.Server.TrustedProxies))
	}
	trustForwardedProto := len(config.Server.TrustedProxies) > 0
	engine.Use(
		logger.RecoveryMiddleware(),
		middleware.BlockUnsafeHTTPMethods(),
		logger.RequestIDMiddleware(),
		middleware.MaxRequestBodyBytes(maxBodyBytes),
		middleware.SecurityHeadersMiddleware(config.Security.HSTSMaxAgeSeconds, trustForwardedProto),
		logger.AccessLogMiddlewareWithOptions(logger.AccessLogOptions{
			SlowRequestThreshold: time.Duration(config.Logging.SlowRequestMS) * time.Millisecond,
		}),
	)
	logger.S().Info("Gin engine created with custom logging middlewares")

	// 配置 CORS
	logger.S().Info("configuring CORS")
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Lang", "Accept-Language"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	logger.S().Info("CORS configured")

	if ipGuard.Enabled() {
		engine.Use(ipGuard.Middleware())
		logger.S().Info("ip blacklist middleware enabled")
	}

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

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.S().Info("shutting down server")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.S().Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	if metricsSrv != nil {
		mDur := 5 * time.Second
		if shutdownTimeout < mDur {
			mDur = shutdownTimeout
		}
		mctx, mcancel := context.WithTimeout(context.Background(), mDur)
		if err := metricsSrv.Shutdown(mctx); err != nil {
			logger.S().Warn("metrics server shutdown", "error", err)
		}
		mcancel()
	}

	// 关闭服务
	reportService.Close()
	if pulsarConsumer != nil {
		pulsarConsumer.Close()
	}
	if pulsarProducer != nil {
		pulsarProducer.Close()
	}
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
