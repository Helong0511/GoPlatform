package util

import (
	"fmt"
	"strings"
	"time"
)

// ValidateConfig 验证配置文件
func (c *Config) ValidateConfig() error {
	if c.environmentFromFile != "" &&
		IsProductionEnvironment(c.environmentFromFile) &&
		!IsProductionEnvironment(c.Environment) {
		return fmt.Errorf(
			"environment: config file sets production but effective environment is %q (ENVIRONMENT override not allowed to downgrade)",
			c.Environment,
		)
	}

	// 验证服务器配置
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d, must be between 1 and 65535", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if c.Server.MaxRequestBodyMB < 0 || c.Server.MaxRequestBodyMB > 512 {
		return fmt.Errorf("server max_request_body_mb must be between 0 and 512 (0 uses default after SetDefaults)")
	}

	if strings.TrimSpace(c.Server.ShutdownTimeout) != "" {
		if _, err := time.ParseDuration(strings.TrimSpace(c.Server.ShutdownTimeout)); err != nil {
			return fmt.Errorf("invalid server shutdown_timeout: %w", err)
		}
	}

	if c.Security.IPBlacklist.Enabled {
		if c.Security.IPBlacklist.StrikesBeforeBan < 0 {
			return fmt.Errorf("security.ip_blacklist.strikes_before_ban must be >= 0")
		}
		if c.Security.IPBlacklist.StrikeWindow != "" {
			if _, err := time.ParseDuration(c.Security.IPBlacklist.StrikeWindow); err != nil {
				return fmt.Errorf("security.ip_blacklist.strike_window: %w", err)
			}
		}
		if c.Security.IPBlacklist.BanDuration != "" {
			if _, err := time.ParseDuration(c.Security.IPBlacklist.BanDuration); err != nil {
				return fmt.Errorf("security.ip_blacklist.ban_duration: %w", err)
			}
		}
		if c.Security.IPBlacklist.CrawlerStrikeWeight < 0 {
			return fmt.Errorf("security.ip_blacklist.crawler_strike_weight must be >= 0")
		}
	}
	if c.Security.HSTSMaxAgeSeconds < 0 || c.Security.HSTSMaxAgeSeconds > 63072000 {
		return fmt.Errorf("security hsts_max_age_seconds must be between 0 and 63072000")
	}

	// 验证数据库配置
	if c.Database.Driver != "postgres" && c.Database.Driver != "mysql" {
		return fmt.Errorf("invalid database driver: %s, must be postgres or mysql", c.Database.Driver)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}

	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d, must be between 1 and 65535", c.Database.Port)
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be greater than 0")
	}

	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns cannot be negative")
	}

	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot be greater than max_open_conns")
	}

	// 验证Redis配置
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host cannot be empty")
	}

	if c.Redis.Port <= 0 || c.Redis.Port > 65535 {
		return fmt.Errorf("invalid redis port: %d, must be between 1 and 65535", c.Redis.Port)
	}

	if c.Redis.DB < 0 || c.Redis.DB > 15 {
		return fmt.Errorf("invalid redis db: %d, must be between 0 and 15", c.Redis.DB)
	}

	// 验证 Pulsar：未启用（URL 为空）时不强制填写 topic/subscription，便于本地与无消息队列部署
	if strings.TrimSpace(c.Pulsar.URL) != "" {
		if strings.TrimSpace(c.Pulsar.Topic) == "" {
			return fmt.Errorf("pulsar topic cannot be empty when pulsar url is set")
		}
		if strings.TrimSpace(c.Pulsar.Subscription) == "" {
			return fmt.Errorf("pulsar subscription cannot be empty when pulsar url is set")
		}
	}

	// 验证JWT配置
	if c.JWT.ExpiresIn == "" {
		return fmt.Errorf("jwt expires_in cannot be empty")
	}

	// 验证监控配置
	if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
		return fmt.Errorf("invalid metrics port: %d, must be between 1 and 65535", c.Metrics.Port)
	}

	// 验证日志配置
	if c.Logging.Level == "" {
		return fmt.Errorf("logging level cannot be empty")
	}
	if c.Logging.SlowRequestMS < 0 {
		return fmt.Errorf("logging slow_request_ms cannot be negative")
	}

	// 验证CORS配置
	if len(c.CORS.AllowOrigins) == 0 {
		return fmt.Errorf("cors allow_origins cannot be empty")
	}

	// 验证速率限制配置
	if c.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("rate_limit requests_per_second must be greater than 0")
	}

	if c.RateLimit.Burst <= 0 {
		return fmt.Errorf("rate_limit burst must be greater than 0")
	}

	if c.RateLimit.LoginBurst > 0 && c.RateLimit.LoginRequestsPerSecond <= 0 {
		return fmt.Errorf("rate_limit login_requests_per_second must be greater than 0 when login_burst is set")
	}
	if c.RateLimit.LoginRequestsPerSecond > 0 && c.RateLimit.LoginBurst <= 0 {
		return fmt.Errorf("rate_limit login_burst must be greater than 0 when login_requests_per_second is set")
	}

	if c.RateLimit.RegisterBurst > 0 && c.RateLimit.RegisterRequestsPerSecond <= 0 {
		return fmt.Errorf("rate_limit register_requests_per_second must be greater than 0 when register_burst is set")
	}
	if c.RateLimit.RegisterRequestsPerSecond > 0 && c.RateLimit.RegisterBurst <= 0 {
		return fmt.Errorf("rate_limit register_burst must be greater than 0 when register_requests_per_second is set")
	}

	if c.Auth.MinPasswordLength < 0 || c.Auth.MinPasswordLength > 72 {
		return fmt.Errorf("auth min_password_length must be between 0 and 72")
	}
	if c.Auth.MinPasswordLength > 0 && c.Auth.MinPasswordLength < 6 {
		return fmt.Errorf("auth min_password_length must be at least 6 when set")
	}

	if IsStrictHardeningEnvironment(c.Environment) {
		envLabel := strings.TrimSpace(c.Environment)
		for _, o := range c.CORS.AllowOrigins {
			if strings.TrimSpace(o) == "*" {
				return fmt.Errorf("%s: cors allow_origins cannot contain wildcard '*'; specify explicit origins", envLabel)
			}
		}
		driver := strings.ToLower(strings.TrimSpace(c.Database.Driver))
		sm := strings.ToLower(strings.TrimSpace(c.Database.SSLMode))
		switch driver {
		case "postgres":
			if sm == "" || sm == "disable" {
				return fmt.Errorf("%s: postgres sslmode must not be disable or empty; use require or verify-full etc.", envLabel)
			}
		case "mysql":
			if sm == "" || sm == "disable" || sm == "false" || sm == "0" {
				return fmt.Errorf("%s: mysql sslmode/tls must be enabled (e.g. true, require, preferred); got %q", envLabel, c.Database.SSLMode)
			}
		}
		if strings.TrimSpace(c.JWT.Issuer) == "" || strings.TrimSpace(c.JWT.Audience) == "" {
			return fmt.Errorf("%s: jwt issuer and audience must be set", envLabel)
		}
		if strings.TrimSpace(c.Metrics.BearerToken) == "" {
			return fmt.Errorf("%s: metrics bearer_token must be set", envLabel)
		}
		if c.PublicRegisterEnabled() {
			return fmt.Errorf("%s: public registration must be disabled", envLabel)
		}
		if c.Auth.MinPasswordLength > 0 && c.Auth.MinPasswordLength < 8 {
			return fmt.Errorf("%s: auth min_password_length must be at least 8", envLabel)
		}
		if strings.TrimSpace(c.Database.Password) == "" || looksLikePlaceholder(c.Database.Password) {
			return fmt.Errorf("%s: database password must be set to a real secret (config or DB_PASSWORD env)", envLabel)
		}
		if strings.TrimSpace(c.Redis.Password) == "" || looksLikePlaceholder(c.Redis.Password) {
			return fmt.Errorf("%s: redis password must be set to a real secret (config or REDIS_PASSWORD env)", envLabel)
		}
		if sec := effectiveEnvOrConfig(EnvJWTSecret, c.JWT.Secret); sec == "" || looksLikePlaceholder(sec) {
			return fmt.Errorf("%s: jwt secret must be set to a real secret (config or JWT_SECRET env)", envLabel)
		}
		if sec := effectiveEnvOrConfig(EnvAESSecretKey, c.Security.AESSecret); sec == "" || looksLikePlaceholder(sec) {
			return fmt.Errorf("%s: aes secret must be set to a real secret (config or AES_SECRET_KEY env)", envLabel)
		}
		if tok := effectiveEnvOrConfig(EnvMetricsBearerToken, c.Metrics.BearerToken); tok == "" || looksLikePlaceholder(tok) {
			return fmt.Errorf("%s: metrics bearer_token must be set to a real secret (config or METRICS_BEARER_TOKEN env)", envLabel)
		}
	}

	if IsProductionEnvironment(c.Environment) {
		host := strings.TrimSpace(c.Server.Host)
		if host == "" || host == "0.0.0.0" {
			return fmt.Errorf("production: server.host should be 127.0.0.1 when behind reverse proxy, got %q", c.Server.Host)
		}
	}

	if c.Email.Enabled {
		if c.Email.Host == "" {
			return fmt.Errorf("email host cannot be empty when email is enabled")
		}
		if c.Email.Port <= 0 || c.Email.Port > 65535 {
			return fmt.Errorf("invalid email port: %d, must be between 1 and 65535", c.Email.Port)
		}
		if c.Email.Username == "" {
			return fmt.Errorf("email username cannot be empty when email is enabled")
		}
		if c.Email.Password == "" {
			return fmt.Errorf("email password cannot be empty when email is enabled")
		}
		if c.Email.From == "" {
			return fmt.Errorf("email from cannot be empty when email is enabled")
		}
	}

	return nil
}

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	// 默认值设置
	if c.Environment == "" {
		c.Environment = "development"
	}

	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}

	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

	if c.Server.Timeout == "" {
		c.Server.Timeout = "30s"
	}

	if c.Server.MaxRequestBodyMB == 0 {
		c.Server.MaxRequestBodyMB = 10
	}
	if c.Server.ShutdownTimeout == "" {
		c.Server.ShutdownTimeout = "15s"
	}

	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}

	if c.Database.Driver == "" {
		c.Database.Driver = "postgres"
	}

	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}

	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}

	if c.Database.ConnMaxLifetime == "" {
		c.Database.ConnMaxLifetime = "5m"
	}

	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 10
	}

	if c.Redis.DialTimeout == "" {
		c.Redis.DialTimeout = "5s"
	}

	if c.Redis.ReadTimeout == "" {
		c.Redis.ReadTimeout = "3s"
	}

	if c.Redis.WriteTimeout == "" {
		c.Redis.WriteTimeout = "3s"
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}

	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.SlowRequestMS == 0 {
		c.Logging.SlowRequestMS = 300
	}

	if len(c.CORS.AllowOrigins) == 0 {
		c.CORS.AllowOrigins = []string{"*"}
	}

	// 速率限制默认值
	if c.RateLimit.RequestsPerSecond == 0 {
		c.RateLimit.RequestsPerSecond = 10.0
	}

	if c.RateLimit.Burst == 0 {
		c.RateLimit.Burst = 50
	}

	// JWT 默认值
	if c.JWT.ExpiresIn == "" {
		c.JWT.ExpiresIn = "1h"
	}
	if strings.TrimSpace(c.JWT.Issuer) == "" {
		c.JWT.Issuer = "ad-platform"
	}
	if strings.TrimSpace(c.JWT.Audience) == "" {
		c.JWT.Audience = "ad-platform-api"
	}

	if c.Email.Port == 0 {
		c.Email.Port = 587
	}
	if c.Email.From == "" && c.Email.Username != "" {
		c.Email.From = c.Email.Username
	}

	if c.Auth.MinPasswordLength == 0 {
		c.Auth.MinPasswordLength = 8
	}

	if IsStrictHardeningEnvironment(c.Environment) {
		if c.RateLimit.LoginRequestsPerSecond <= 0 {
			c.RateLimit.LoginRequestsPerSecond = 1
			c.RateLimit.LoginBurst = 5
		}
		if c.Metrics.Host == "" || c.Metrics.Host == "0.0.0.0" {
			c.Metrics.Host = "127.0.0.1"
		}
	}

	if c.Security.IPBlacklist.StrikeWindow == "" {
		c.Security.IPBlacklist.StrikeWindow = "10m"
	}
	if c.Security.IPBlacklist.BanDuration == "" {
		c.Security.IPBlacklist.BanDuration = "24h"
	}
	if c.Security.IPBlacklist.StrikesBeforeBan == 0 {
		c.Security.IPBlacklist.StrikesBeforeBan = 5
	}
	if c.Security.IPBlacklist.CrawlerStrikeWeight == 0 {
		c.Security.IPBlacklist.CrawlerStrikeWeight = 2
	}

	// 公开注册默认值由 PublicRegisterEnabled() 按环境动态决定。
}
