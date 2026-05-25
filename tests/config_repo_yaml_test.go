package tests

import (
	"os"
	"path/filepath"
	"testing"

	"ad-platform/pkg/util"
)

func repoConfigPath(t *testing.T) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "config", "config.yaml"),
		filepath.Join("config", "config.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("config/config.yaml not found from tests package")
	return ""
}

func TestRepoConfigYAMLDevelopmentLoadsAndValidates(t *testing.T) {
	path := repoConfigPath(t)
	cfg, err := util.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Environment != "development" {
		t.Fatalf("expected development in repo config, got %q", cfg.Environment)
	}
	cfg.SetDefaults()
	if err := cfg.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig development: %v", err)
	}
	if cfg.JWT.Issuer == "" || cfg.JWT.Audience == "" {
		t.Fatalf("expected jwt issuer/audience defaults after SetDefaults")
	}
}

func TestRepoConfigYAMLProductionTemplateValidatesWithSecrets(t *testing.T) {
	path := repoConfigPath(t)
	cfg, err := util.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Environment = "production"
	cfg.Database.SSLMode = "require"
	cfg.Database.Password = "db-pass"
	cfg.Redis.Password = "redis-pass"
	cfg.Metrics.BearerToken = "metrics-token"
	cfg.JWT.Secret = "12345678901234567890123456789012"
	cfg.Security.AESSecret = "awwYDrn0ojn2p7y2hNL/kg=="
	cfg.JWT.Issuer = "ad-platform"
	cfg.JWT.Audience = "ad-platform-api"
	cfg.Auth.AllowPublicRegister = boolPtr(false)
	cfg.Server.Host = "127.0.0.1"
	cfg.SetDefaults()
	if err := cfg.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig production template: %v", err)
	}
}

func TestProductionRequiresRedisAndDBPassword(t *testing.T) {
	base := func() *util.Config {
		return &util.Config{
			Environment: "production",
			Server:      util.ServerConfig{Host: "127.0.0.1", Port: 8080},
			Database: util.DatabaseConfig{
				Driver: "postgres", Host: "localhost", Port: 5432,
				User: "u", DBName: "d", SSLMode: "require",
				Password: "db-pw", MaxOpenConns: 10, MaxIdleConns: 2,
			},
			Redis: util.RedisConfig{
				Host: "localhost", Port: 6379, DB: 0, Password: "redis-pw",
			},
			Pulsar: util.PulsarConfig{},
			JWT: util.JWTConfig{
				ExpiresIn: "1h", Issuer: "ad-platform", Audience: "ad-platform-api",
			},
			Metrics: util.MetricsConfig{Port: 9090, Path: "/metrics", BearerToken: "tok"},
			Logging: util.LoggingConfig{Level: "info", Format: "json"},
			CORS:    util.CORSConfig{AllowOrigins: []string{"https://app.example.com"}},
			RateLimit: util.RateLimitConfig{RequestsPerSecond: 10, Burst: 20},
		}
	}

	t.Run("missing redis password", func(t *testing.T) {
		cfg := base()
		cfg.Redis.Password = ""
		if err := cfg.ValidateConfig(); err == nil {
			t.Fatal("expected error for empty redis password in production")
		}
	})

	t.Run("missing database password", func(t *testing.T) {
		cfg := base()
		cfg.Database.Password = ""
		if err := cfg.ValidateConfig(); err == nil {
			t.Fatal("expected error for empty database password in production")
		}
	})
}

func boolPtr(v bool) *bool { return &v }
