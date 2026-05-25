package tests

import (
	"os"
	"path/filepath"
	"testing"

	"ad-platform/pkg/util"
)

func TestLoadDotEnvFilesDoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("DOTENV_TEST_KEY", "from-os")
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".env")
	if err := os.WriteFile(path, []byte("DOTENV_TEST_KEY=from-file\nDOTENV_OTHER=only-file\n"), 0644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	loaded, err := util.LoadDotEnvFiles(path)
	if err != nil {
		t.Fatalf("LoadDotEnvFiles: %v", err)
	}
	if loaded != path {
		t.Fatalf("expected loaded %q, got %q", path, loaded)
	}
	if os.Getenv("DOTENV_TEST_KEY") != "from-os" {
		t.Fatalf("must not override existing env")
	}
	if os.Getenv("DOTENV_OTHER") != "only-file" {
		t.Fatalf("expected DOTENV_OTHER from file, got %q", os.Getenv("DOTENV_OTHER"))
	}
}

func TestStrictHardeningIncludesStaging(t *testing.T) {
	if !util.IsStrictHardeningEnvironment("staging") {
		t.Fatal("staging should use strict hardening")
	}
	if util.IsStrictHardeningEnvironment("development") {
		t.Fatal("development should not use strict hardening")
	}
}

func TestStagingRejectsDisabledPostgresSSL(t *testing.T) {
	cfg := &util.Config{
		Environment: "staging",
		Server:      util.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: util.DatabaseConfig{
			Driver: "postgres", Host: "localhost", Port: 5432,
			User: "u", Password: "real-db-pass", DBName: "d", SSLMode: "disable",
			MaxOpenConns: 10, MaxIdleConns: 2,
		},
		Redis: util.RedisConfig{Host: "localhost", Port: 6379, DB: 0, Password: "redis-pass"},
		Pulsar: util.PulsarConfig{},
		JWT: util.JWTConfig{
			Secret: "12345678901234567890123456789012", ExpiresIn: "1h",
			Issuer: "ad-platform", Audience: "ad-platform-api",
		},
		Security: util.SecurityConfig{AESSecret: "awwYDrn0ojn2p7y2hNL/kg=="},
		Metrics:  util.MetricsConfig{Port: 9090, Path: "/metrics", BearerToken: "metrics-tok"},
		Logging:  util.LoggingConfig{Level: "info", Format: "json"},
		CORS:     util.CORSConfig{AllowOrigins: []string{"https://app.example.com"}},
		RateLimit: util.RateLimitConfig{RequestsPerSecond: 10, Burst: 20},
	}
	if err := cfg.ValidateConfig(); err == nil {
		t.Fatal("expected staging to reject postgres sslmode=disable")
	}
}
