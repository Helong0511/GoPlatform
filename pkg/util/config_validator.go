package util

import "fmt"

// ValidateConfig 验证配置文件
func (c *Config) ValidateConfig() error {
	// 验证服务器配置
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d, must be between 1 and 65535", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
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

	// 验证Pulsar配置
	if c.Pulsar.URL == "" {
		return fmt.Errorf("pulsar URL cannot be empty")
	}

	if c.Pulsar.Topic == "" {
		return fmt.Errorf("pulsar topic cannot be empty")
	}

	if c.Pulsar.Subscription == "" {
		return fmt.Errorf("pulsar subscription cannot be empty")
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

	if c.Email.Port == 0 {
		c.Email.Port = 587
	}
	if c.Email.From == "" && c.Email.Username != "" {
		c.Email.From = c.Email.Username
	}

	// 公开注册默认值由 PublicRegisterEnabled() 按环境动态决定。
}
