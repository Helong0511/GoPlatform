package util

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string          `yaml:"environment" json:"environment"`
	// environmentFromFile 为 YAML 中的 environment，用于检测 ENVIRONMENT 环境变量是否将生产降级。
	environmentFromFile string
	Server      ServerConfig    `yaml:"server" json:"server"`
	Auth        AuthConfig      `yaml:"auth" json:"auth"`
	Email       EmailConfig     `yaml:"email" json:"email"`
	Database    DatabaseConfig  `yaml:"database" json:"database"`
	Redis       RedisConfig     `yaml:"redis" json:"redis"`
	Pulsar      PulsarConfig    `yaml:"pulsar" json:"pulsar"`
	JWT         JWTConfig       `yaml:"jwt" json:"jwt"`
	Security    SecurityConfig  `yaml:"security" json:"security"`
	Metrics     MetricsConfig   `yaml:"metrics" json:"metrics"`
	Logging     LoggingConfig   `yaml:"logging" json:"logging"`
	CORS        CORSConfig      `yaml:"cors" json:"cors"`
	RateLimit   RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
}

// AuthConfig 认证/注册相关开关
type AuthConfig struct {
	// AllowPublicRegister:
	// - true: 强制开启公开注册
	// - false: 强制关闭公开注册
	// - nil: 按环境默认（development/local/dev 默认开启，其他默认关闭）
	AllowPublicRegister *bool `yaml:"allow_public_register" json:"allow_public_register"`
	// MinPasswordLength 管理员创建/修改用户时的明文密码最短字符数（Unicode 字符计数）
	MinPasswordLength int `yaml:"min_password_length" json:"min_password_length"`
}

type EmailConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	From     string `yaml:"from" json:"from"`
}

type ServerConfig struct {
	Port             int      `yaml:"port" json:"port"`
	Host             string   `yaml:"host" json:"host"`
	Timeout          string   `yaml:"timeout" json:"timeout"`
	TrustedProxies   []string `yaml:"trusted_proxies" json:"trusted_proxies"`
	MaxRequestBodyMB int      `yaml:"max_request_body_mb" json:"max_request_body_mb"`
	ShutdownTimeout  string   `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Driver          string           `yaml:"driver" json:"driver"`
	Host            string           `yaml:"host" json:"host"`
	Port            int              `yaml:"port" json:"port"`
	User            string           `yaml:"user" json:"user"`
	Password        string           `yaml:"password" json:"password"`
	DBName          string           `yaml:"dbname" json:"dbname"`
	SSLMode         string           `yaml:"sslmode" json:"sslmode"`
	MaxOpenConns    int              `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int              `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime string           `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime string           `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
	Cluster         []DatabaseConfig `yaml:"cluster" json:"cluster,omitempty"`
}

type RedisConfig struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	Password        string        `yaml:"password" json:"password"`
	DB              int           `yaml:"db" json:"db"`
	PoolSize        int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConns    int           `yaml:"min_idle_conns" json:"min_idle_conns"`
	DialTimeout     string        `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout     string        `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    string        `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout     string        `yaml:"idle_timeout" json:"idle_timeout"`
	MaxRetries      int           `yaml:"max_retries" json:"max_retries"`
	MinRetryBackoff string        `yaml:"min_retry_backoff" json:"min_retry_backoff"`
	MaxRetryBackoff string        `yaml:"max_retry_backoff" json:"max_retry_backoff"`
	Cluster         []RedisConfig `yaml:"cluster" json:"cluster,omitempty"`
}

type PulsarConfig struct {
	URL            string `yaml:"url" json:"url"`
	Topic          string `yaml:"topic" json:"topic"`
	Subscription   string `yaml:"subscription" json:"subscription"`
	MaxMessageSize int    `yaml:"max_message_size" json:"max_message_size"`
}

type JWTConfig struct {
	Secret    string `yaml:"secret" json:"secret"`
	ExpiresIn string `yaml:"expires_in" json:"expires_in"`
	Issuer    string `yaml:"issuer" json:"issuer"`
	Audience  string `yaml:"audience" json:"audience"`
}

type SecurityConfig struct {
	RustLibPath       string            `yaml:"rust_lib_path" json:"rust_lib_path"`
	AESSecret         string            `yaml:"aes_secret" json:"aes_secret"`
	HSTSMaxAgeSeconds int               `yaml:"hsts_max_age_seconds" json:"hsts_max_age_seconds"`
	IPBlacklist       IPBlacklistConfig `yaml:"ip_blacklist" json:"ip_blacklist"`
}

// IPBlacklistConfig 自动将爬取/访问敏感资源的 IP 加入黑名单。
type IPBlacklistConfig struct {
	Enabled                   bool     `yaml:"enabled" json:"enabled"`
	BanDuration               string   `yaml:"ban_duration" json:"ban_duration"`
	StrikeWindow              string   `yaml:"strike_window" json:"strike_window"`
	StrikesBeforeBan          int      `yaml:"strikes_before_ban" json:"strikes_before_ban"`
	BlockCrawlerUserAgent     bool     `yaml:"block_crawler_user_agent" json:"block_crawler_user_agent"`
	CrawlerStrikeWeight       int      `yaml:"crawler_strike_weight" json:"crawler_strike_weight"`
	StaticIPs                 []string `yaml:"static_ips" json:"static_ips"`
	StaticWhitelistIPs        []string `yaml:"static_whitelist_ips" json:"static_whitelist_ips"`
}

type MetricsConfig struct {
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Path        string `yaml:"path" json:"path"`
	BearerToken string `yaml:"bearer_token" json:"bearer_token"`
}

type LoggingConfig struct {
	Level         string `yaml:"level" json:"level"`
	Format        string `yaml:"format" json:"format"`
	SlowRequestMS int    `yaml:"slow_request_ms" json:"slow_request_ms"`
}

type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins" json:"allow_origins"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second" json:"requests_per_second"`
	Burst             int     `yaml:"burst" json:"burst"`
	// LoginRequestsPerSecond / LoginBurst：仅作用于 POST /api/login；二者须同时大于 0 才启用，默认 0 表示不额外限制（仍走全局限流）
	LoginRequestsPerSecond float64 `yaml:"login_requests_per_second" json:"login_requests_per_second"`
	LoginBurst             int     `yaml:"login_burst" json:"login_burst"`
	// RegisterRequestsPerSecond / RegisterBurst：仅作用于 POST /api/register；成对启用规则同登录
	RegisterRequestsPerSecond float64 `yaml:"register_requests_per_second" json:"register_requests_per_second"`
	RegisterBurst             int     `yaml:"register_burst" json:"register_burst"`
}

// LoadConfig 加载配置文件并支持环境变量覆盖
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}
	config.environmentFromFile = strings.TrimSpace(config.Environment)

	// 从环境变量覆盖配置，避免硬编码凭证
	if envDBHost := GetEnvTrim(EnvDBHost); envDBHost != "" {
		config.Database.Host = envDBHost
		// 同时更新集群配置
		for i := range config.Database.Cluster {
			config.Database.Cluster[i].Host = envDBHost
		}
	}
	if envDBDriver := GetEnvTrim(EnvDBDriver); envDBDriver != "" {
		config.Database.Driver = strings.ToLower(envDBDriver)
		for i := range config.Database.Cluster {
			config.Database.Cluster[i].Driver = strings.ToLower(envDBDriver)
		}
	}
	if envDBPort := GetEnvTrim(EnvDBPort); envDBPort != "" {
		if port, err := strconv.Atoi(envDBPort); err == nil {
			config.Database.Port = port
			// 同时更新集群配置
			for i := range config.Database.Cluster {
				config.Database.Cluster[i].Port = port
			}
		}
	}
	if envDBUser := GetEnvTrim(EnvDBUser); envDBUser != "" {
		config.Database.User = envDBUser
		// 同时更新集群配置
		for i := range config.Database.Cluster {
			config.Database.Cluster[i].User = envDBUser
		}
	}
	if envDBPassword := GetEnvTrim(EnvDBPassword); envDBPassword != "" {
		config.Database.Password = envDBPassword
		// 同时更新集群配置
		for i := range config.Database.Cluster {
			config.Database.Cluster[i].Password = envDBPassword
		}
	}
	if envDBName := GetEnvTrim(EnvDBName); envDBName != "" {
		config.Database.DBName = envDBName
		// 同时更新集群配置
		for i := range config.Database.Cluster {
			config.Database.Cluster[i].DBName = envDBName
		}
	}
	if envJWTSecret := GetEnvTrim(EnvJWTSecret); envJWTSecret != "" {
		config.JWT.Secret = envJWTSecret
	}
	if envRedisPassword := GetEnvTrim(EnvRedisPassword); envRedisPassword != "" {
		config.Redis.Password = envRedisPassword
	}
	if envAESSecret := GetEnvTrim(EnvAESSecretKey); envAESSecret != "" {
		config.Security.AESSecret = envAESSecret
	}
	if envEnvironment := GetEnvTrim(EnvEnvironment); envEnvironment != "" {
		config.Environment = envEnvironment
	}
	if v := GetEnvTrim(EnvAuthAllowPublicRegister); v != "" {
		envAllow := parseEnvBool(v)
		config.Auth.AllowPublicRegister = &envAllow
	}
	if v := GetEnvTrim(EnvEmailEnabled); v != "" {
		config.Email.Enabled = parseEnvBool(v)
	}
	if v := GetEnvTrim(EnvEmailHost); v != "" {
		config.Email.Host = v
	}
	if v := GetEnvTrim(EnvEmailPort); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			config.Email.Port = p
		}
	}
	if v := GetEnvTrim(EnvEmailUsername); v != "" {
		config.Email.Username = v
	}
	if v := GetEnvTrim(EnvEmailPassword); v != "" {
		config.Email.Password = v
	}
	if v := GetEnvTrim(EnvEmailFrom); v != "" {
		config.Email.From = v
	}
	if v := GetEnvTrim(EnvMetricsBearerToken); v != "" {
		config.Metrics.BearerToken = v
	}
	if v := GetEnvTrim(EnvTrustedProxies); v != "" {
		var proxies []string
		for _, p := range strings.Split(v, ",") {
			if t := strings.TrimSpace(p); t != "" {
				proxies = append(proxies, t)
			}
		}
		if len(proxies) > 0 {
			config.Server.TrustedProxies = proxies
		}
	}

	return &config, nil
}

// PublicRegisterEnabled 返回最终公开注册开关。
func (c *Config) PublicRegisterEnabled() bool {
	if c.Auth.AllowPublicRegister != nil {
		return *c.Auth.AllowPublicRegister
	}

	env := strings.ToLower(strings.TrimSpace(c.Environment))
	return env == "development" || env == "dev" || env == "local"
}

func parseEnvBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}
