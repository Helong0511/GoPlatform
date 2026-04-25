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
