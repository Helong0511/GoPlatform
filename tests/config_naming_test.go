package tests

import (
	"os"
	"path/filepath"
	"testing"

	"ad-platform/pkg/util"
)

func TestLoadConfigParsesSnakeCaseSections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  host: 127.0.0.1
  port: 8080
  timeout: 15s
database:
  driver: mysql
  host: localhost
  port: 5432
  user: test_user
  password: test_pwd
  dbname: test_db
  sslmode: disable
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: 1m
  conn_max_idle_time: 1m
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 8
  min_idle_conns: 2
  dial_timeout: 1s
  read_timeout: 1s
  write_timeout: 1s
  idle_timeout: 1m
  max_retries: 2
  min_retry_backoff: 50ms
  max_retry_backoff: 500ms
  cluster:
    - host: 127.0.0.2
      port: 6380
      db: 1
pulsar:
  url: pulsar://localhost:6650
  topic: persistent://public/default/test
  subscription: test_sub
  max_message_size: 1024
jwt:
  secret: test_secret
  expires_in: 1h
security:
  rust_lib_path: ./rust-security/target/debug/rust_security.dll
  aes_secret: test_aes_secret
metrics:
  port: 9090
  path: /metrics
logging:
  level: info
  format: json
cors:
  allow_origins: ["*"]
rate_limit:
  requests_per_second: 10
  burst: 20
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := util.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected main server port to be parsed, got main=%d", cfg.Server.Port)
	}
	if cfg.Database.Driver != "mysql" {
		t.Fatalf("expected database driver mysql, got %s", cfg.Database.Driver)
	}
	if len(cfg.Redis.Cluster) != 1 {
		t.Fatalf("expected redis cluster to be parsed, got %d", len(cfg.Redis.Cluster))
	}
	if cfg.Redis.Cluster[0].Host != "127.0.0.2" || cfg.Redis.Cluster[0].Port != 6380 {
		t.Fatalf("unexpected redis cluster node: %+v", cfg.Redis.Cluster[0])
	}
}

func TestValidateConfigPulsarOptionalWhenURLBlank(t *testing.T) {
	cfg := &util.Config{
		Server: util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver:       "postgres",
			Host:         "localhost",
			Port:         5432,
			User:         "u",
			DBName:       "d",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		},
		Redis: util.RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		Pulsar: util.PulsarConfig{},
		JWT:    util.JWTConfig{ExpiresIn: "1h"},
		Metrics: util.MetricsConfig{
			Port: 9090,
			Path: "/metrics",
		},
		Logging: util.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		CORS: util.CORSConfig{
			AllowOrigins: []string{"http://localhost"},
		},
		RateLimit: util.RateLimitConfig{
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
	if err := cfg.ValidateConfig(); err != nil {
		t.Fatalf("expected empty pulsar url to validate, got %v", err)
	}
}

func TestValidateConfigPulsarRequiresTopicWhenURLSet(t *testing.T) {
	cfg := &util.Config{
		Server: util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver:       "postgres",
			Host:         "localhost",
			Port:         5432,
			User:         "u",
			DBName:       "d",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		},
		Redis: util.RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		Pulsar: util.PulsarConfig{
			URL:            "pulsar://localhost:6650",
			Topic:          "",
			Subscription:   "sub",
			MaxMessageSize: 1024,
		},
		JWT: util.JWTConfig{ExpiresIn: "1h"},
		Metrics: util.MetricsConfig{
			Port: 9090,
			Path: "/metrics",
		},
		Logging: util.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		CORS: util.CORSConfig{
			AllowOrigins: []string{"http://localhost"},
		},
		RateLimit: util.RateLimitConfig{
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatalf("expected error when pulsar url set but topic empty")
	}
}

func TestValidateConfigLoginRatePairing(t *testing.T) {
	base := func() *util.Config {
		return &util.Config{
			Server: util.ServerConfig{Host: "127.0.0.1", Port: 8080},
			Database: util.DatabaseConfig{
				Driver:       "postgres",
				Host:         "localhost",
				Port:         5432,
				User:         "u",
				DBName:       "d",
				MaxOpenConns: 10,
				MaxIdleConns: 2,
			},
			Redis:  util.RedisConfig{Host: "localhost", Port: 6379, DB: 0},
			Pulsar: util.PulsarConfig{},
			JWT:    util.JWTConfig{ExpiresIn: "1h"},
			Metrics: util.MetricsConfig{
				Port: 9090,
				Path: "/metrics",
			},
			Logging: util.LoggingConfig{Level: "info", Format: "json"},
			CORS:    util.CORSConfig{AllowOrigins: []string{"http://localhost"}},
			RateLimit: util.RateLimitConfig{
				RequestsPerSecond: 10,
				Burst:             20,
			},
		}
	}

	t.Run("both login fields zero is ok", func(t *testing.T) {
		cfg := base()
		if err := cfg.ValidateConfig(); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})

	t.Run("login rps without burst fails", func(t *testing.T) {
		cfg := base()
		cfg.RateLimit.LoginRequestsPerSecond = 2
		if err := cfg.ValidateConfig(); err == nil {
			t.Fatalf("expected error when login_burst missing")
		}
	})

	t.Run("login burst without rps fails", func(t *testing.T) {
		cfg := base()
		cfg.RateLimit.LoginBurst = 5
		if err := cfg.ValidateConfig(); err == nil {
			t.Fatalf("expected error when login_requests_per_second missing")
		}
	})

	t.Run("both login fields set is ok", func(t *testing.T) {
		cfg := base()
		cfg.RateLimit.LoginRequestsPerSecond = 1
		cfg.RateLimit.LoginBurst = 10
		if err := cfg.ValidateConfig(); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})
}

func TestValidateConfigRegisterRatePairing(t *testing.T) {
	base := func() *util.Config {
		return &util.Config{
			Server: util.ServerConfig{Host: "127.0.0.1", Port: 8080},
			Database: util.DatabaseConfig{
				Driver:       "postgres",
				Host:         "localhost",
				Port:         5432,
				User:         "u",
				DBName:       "d",
				MaxOpenConns: 10,
				MaxIdleConns: 2,
			},
			Redis:  util.RedisConfig{Host: "localhost", Port: 6379, DB: 0},
			Pulsar: util.PulsarConfig{},
			JWT:    util.JWTConfig{ExpiresIn: "1h"},
			Metrics: util.MetricsConfig{
				Port: 9090,
				Path: "/metrics",
			},
			Logging: util.LoggingConfig{Level: "info", Format: "json"},
			CORS:    util.CORSConfig{AllowOrigins: []string{"http://localhost"}},
			RateLimit: util.RateLimitConfig{
				RequestsPerSecond: 10,
				Burst:             20,
			},
		}
	}

	t.Run("both register fields zero is ok", func(t *testing.T) {
		cfg := base()
		if err := cfg.ValidateConfig(); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})

	t.Run("register rps without burst fails", func(t *testing.T) {
		cfg := base()
		cfg.RateLimit.RegisterRequestsPerSecond = 0.5
		if err := cfg.ValidateConfig(); err == nil {
			t.Fatalf("expected error when register_burst missing")
		}
	})

	t.Run("register burst without rps fails", func(t *testing.T) {
		cfg := base()
		cfg.RateLimit.RegisterBurst = 5
		if err := cfg.ValidateConfig(); err == nil {
			t.Fatalf("expected error when register_requests_per_second missing")
		}
	})

	t.Run("both register fields set is ok", func(t *testing.T) {
		cfg := base()
		cfg.RateLimit.RegisterRequestsPerSecond = 0.5
		cfg.RateLimit.RegisterBurst = 8
		if err := cfg.ValidateConfig(); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})
}

func TestProductionRejectsPostgresSSLDisable(t *testing.T) {
	cfg := &util.Config{
		Environment: "production",
		Server:      util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver:       "postgres",
			Host:         "localhost",
			Port:         5432,
			User:         "u",
			DBName:       "d",
			SSLMode:      "disable",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		},
		Redis:  util.RedisConfig{Host: "localhost", Port: 6379, DB: 0},
		Pulsar: util.PulsarConfig{},
		JWT: util.JWTConfig{
			ExpiresIn: "1h",
			Issuer:    "ad-platform",
			Audience:  "ad-platform-api",
		},
		Metrics: util.MetricsConfig{
			Port:        9090,
			Path:        "/metrics",
			BearerToken: "metrics-secret",
		},
		Logging: util.LoggingConfig{Level: "info", Format: "json"},
		CORS:    util.CORSConfig{AllowOrigins: []string{"https://app.example.com"}},
		RateLimit: util.RateLimitConfig{
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatalf("expected validation error for postgres sslmode=disable in production")
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	cfg := &util.Config{
		Environment: "production",
		Server:      util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver:       "postgres",
			Host:         "localhost",
			Port:         5432,
			User:         "u",
			DBName:       "d",
			SSLMode:      "require",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		},
		Redis:  util.RedisConfig{Host: "localhost", Port: 6379, DB: 0},
		Pulsar: util.PulsarConfig{},
		JWT: util.JWTConfig{
			ExpiresIn: "1h",
			Issuer:    "ad-platform",
			Audience:  "ad-platform-api",
		},
		Metrics: util.MetricsConfig{
			Port:        9090,
			Path:        "/metrics",
			BearerToken: "metrics-secret",
		},
		Logging: util.LoggingConfig{Level: "info", Format: "json"},
		CORS:    util.CORSConfig{AllowOrigins: []string{"*"}},
		RateLimit: util.RateLimitConfig{
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatalf("expected validation error for wildcard CORS in production")
	}
}

func TestProductionRejectsMySQLSSLDisable(t *testing.T) {
	cfg := &util.Config{
		Environment: "production",
		Server:      util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver:       "mysql",
			Host:         "localhost",
			Port:         3306,
			User:         "u",
			DBName:       "d",
			SSLMode:      "disable",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		},
		Redis:  util.RedisConfig{Host: "localhost", Port: 6379, DB: 0},
		Pulsar: util.PulsarConfig{},
		JWT: util.JWTConfig{
			ExpiresIn: "1h",
			Issuer:    "ad-platform",
			Audience:  "ad-platform-api",
		},
		Metrics: util.MetricsConfig{
			Port:        9090,
			Path:        "/metrics",
			BearerToken: "secret-token",
		},
		Logging: util.LoggingConfig{Level: "info", Format: "json"},
		CORS:    util.CORSConfig{AllowOrigins: []string{"https://app.example.com"}},
		RateLimit: util.RateLimitConfig{
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatalf("expected validation error for mysql sslmode=disable in production")
	}
}

func TestProductionEnvironmentCannotBeDowngradedByEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := `
environment: production
server:
  host: 127.0.0.1
  port: 8080
database:
  driver: postgres
  host: localhost
  port: 5432
  user: u
  password: p
  dbname: d
  sslmode: require
  max_open_conns: 10
  max_idle_conns: 2
redis:
  host: localhost
  port: 6379
jwt:
  secret: s
  expires_in: 1h
  issuer: ad-platform
  audience: ad-platform-api
metrics:
  port: 9090
  path: /metrics
  bearer_token: tok
logging:
  level: info
  format: json
cors:
  allow_origins: ["https://app.example.com"]
rate_limit:
  requests_per_second: 10
  burst: 20
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv(util.EnvEnvironment, "development")
	cfg, err := util.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.SetDefaults()
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatalf("expected error when downgrading production via ENVIRONMENT")
	}
}

func TestJWTDefaultsAfterSetDefaults(t *testing.T) {
	cfg := &util.Config{}
	cfg.SetDefaults()
	if cfg.JWT.Issuer != "ad-platform" || cfg.JWT.Audience != "ad-platform-api" {
		t.Fatalf("unexpected jwt defaults: issuer=%q audience=%q", cfg.JWT.Issuer, cfg.JWT.Audience)
	}
}

func TestProductionRequiresMetricsBearerToken(t *testing.T) {
	cfg := &util.Config{
		Environment: "production",
		Server:      util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver:       "postgres",
			Host:         "localhost",
			Port:         5432,
			User:         "u",
			Password:     "db-secret",
			DBName:       "d",
			SSLMode:      "require",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		},
		Redis: util.RedisConfig{Host: "localhost", Port: 6379, DB: 0, Password: "redis-secret"},
		Pulsar: util.PulsarConfig{},
		JWT: util.JWTConfig{
			ExpiresIn: "1h",
			Issuer:    "ad-platform",
			Audience:  "ad-platform-api",
		},
		Metrics: util.MetricsConfig{
			Port: 9090,
			Path: "/metrics",
		},
		Logging: util.LoggingConfig{Level: "info", Format: "json"},
		CORS:    util.CORSConfig{AllowOrigins: []string{"https://app.example.com"}},
		RateLimit: util.RateLimitConfig{
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatalf("expected validation error when metrics bearer_token is empty in production")
	}
}

func TestPublicRegisterEnabledDefaultsByEnvironment(t *testing.T) {
	devConfig := &util.Config{Environment: "development"}
	devConfig.SetDefaults()
	if !devConfig.PublicRegisterEnabled() {
		t.Fatalf("expected public register enabled by default in development")
	}

	prodConfig := &util.Config{Environment: "production"}
	prodConfig.SetDefaults()
	if prodConfig.PublicRegisterEnabled() {
		t.Fatalf("expected public register disabled by default in production")
	}
}
