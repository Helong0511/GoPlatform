package util

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string          `yaml:"environment"`
	Server      ServerConfig    `yaml:"server"`
	Auth        AuthConfig      `yaml:"auth"`
	Email       EmailConfig     `yaml:"email"`
	Database    DatabaseConfig  `yaml:"database"`
	Redis       RedisConfig     `yaml:"redis"`
	Pulsar      PulsarConfig    `yaml:"pulsar"`
	JWT         JWTConfig       `yaml:"jwt"`
	Security    SecurityConfig  `yaml:"security"`
	Metrics     MetricsConfig   `yaml:"metrics"`
	Logging     LoggingConfig   `yaml:"logging"`
	CORS        CORSConfig      `yaml:"cors"`
	RateLimit   RateLimitConfig `yaml:"rate_limit"`
}

// AuthConfig 认证/注册相关开关
type AuthConfig struct {
	// AllowPublicRegister:
	// - true: 强制开启公开注册
	// - false: 强制关闭公开注册
	// - nil: 按环境默认（development/local/dev 默认开启，其他默认关闭）
	AllowPublicRegister *bool `yaml:"allow_public_register"`
}

type EmailConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type ServerConfig struct {
	Port    int    `yaml:"port"`
	Host    string `yaml:"host"`
	Timeout string `yaml:"timeout"`
}

type DatabaseConfig struct {
	Driver          string           `yaml:"driver"`
	Host            string           `yaml:"host"`
	Port            int              `yaml:"port"`
	User            string           `yaml:"user"`
	Password        string           `yaml:"password"`
	DBName          string           `yaml:"dbname"`
	SSLMode         string           `yaml:"sslmode"`
	MaxOpenConns    int              `yaml:"max_open_conns"`
	MaxIdleConns    int              `yaml:"max_idle_conns"`
	ConnMaxLifetime string           `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime string           `yaml:"conn_max_idle_time"`
	Cluster         []DatabaseConfig `yaml:"cluster"`
}

type RedisConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Password        string        `yaml:"password"`
	DB              int           `yaml:"db"`
	PoolSize        int           `yaml:"pool_size"`
	MinIdleConns    int           `yaml:"min_idle_conns"`
	DialTimeout     string        `yaml:"dial_timeout"`
	ReadTimeout     string        `yaml:"read_timeout"`
	WriteTimeout    string        `yaml:"write_timeout"`
	IdleTimeout     string        `yaml:"idle_timeout"`
	MaxRetries      int           `yaml:"max_retries"`
	MinRetryBackoff string        `yaml:"min_retry_backoff"`
	MaxRetryBackoff string        `yaml:"max_retry_backoff"`
	Cluster         []RedisConfig `yaml:"cluster"`
}

type PulsarConfig struct {
	URL            string `yaml:"url"`
	Topic          string `yaml:"topic"`
	Subscription   string `yaml:"subscription"`
	MaxMessageSize int    `yaml:"max_message_size"`
}

type JWTConfig struct {
	Secret    string `yaml:"secret"`
	ExpiresIn string `yaml:"expires_in"`
}

type SecurityConfig struct {
	RustLibPath string `yaml:"rust_lib_path"`
	AESSecret   string `yaml:"aes_secret"`
}

type MetricsConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
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
