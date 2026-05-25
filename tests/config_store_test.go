package tests

import (
	"os"
	"path/filepath"
	"testing"

	"ad-platform/pkg/middleware"
	"ad-platform/pkg/util"
)

func TestConfigStoreAdminGetAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := `environment: development
server:
  port: 8960
  host: 127.0.0.1
database:
  driver: postgres
  host: localhost
  port: 5432
  user: postgres
  password: ""
  dbname: ad_platform
  sslmode: disable
redis:
  host: localhost
  port: 6379
jwt:
  secret: "12345678901234567890123456789012"
  expires_in: 1h
security:
  aes_secret: "1234567890123456"
  ip_blacklist:
    enabled: true
    ban_duration: 24h
    strike_window: 10m
    strikes_before_ban: 5
metrics:
  port: 9090
logging:
  level: info
cors:
  allow_origins: ["*"]
rate_limit:
  requests_per_second: 10
  burst: 50
`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	store := util.NewConfigStore(path, "")
	view, err := store.AdminGet()
	if err != nil {
		t.Fatal(err)
	}
	if view.File.JWT.Secret != util.SecretPlaceholder {
		t.Fatalf("expected masked jwt secret, got %q", view.File.JWT.Secret)
	}

	incoming, err := util.LoadConfigYAMLOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	incoming.Security.IPBlacklist.Enabled = false
	result, err := store.AdminSave(incoming, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Saved {
		t.Fatal("expected saved")
	}

	reloaded, err := util.LoadConfigYAMLOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Security.IPBlacklist.Enabled {
		t.Fatal("expected ip blacklist disabled in file")
	}
}

func TestConfigStoreHotReloadIPGuard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := `environment: development
server:
  port: 8960
  host: 127.0.0.1
database:
  driver: postgres
  host: localhost
  port: 5432
  user: u
  password: p
  dbname: d
  sslmode: disable
redis:
  host: localhost
  port: 6379
jwt:
  secret: "12345678901234567890123456789012"
  expires_in: 1h
security:
  aes_secret: "1234567890123456"
  ip_blacklist:
    enabled: false
    ban_duration: 24h
    strike_window: 10m
    strikes_before_ban: 5
metrics:
  port: 9090
logging:
  level: info
cors:
  allow_origins: ["*"]
rate_limit:
  requests_per_second: 10
  burst: 50
`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	guard := middleware.NewIPGuard(middleware.IPGuardOptions{Enabled: false}, nil)
	store := util.NewConfigStore(path, "")
	store.RegisterHotReload("security.ip_blacklist", func(cfg *util.Config) error {
		opts, err := middleware.ParseIPGuardOptions(middleware.IPBlacklistYAML{
			Enabled:          cfg.Security.IPBlacklist.Enabled,
			BanDuration:      cfg.Security.IPBlacklist.BanDuration,
			StrikeWindow:     cfg.Security.IPBlacklist.StrikeWindow,
			StrikesBeforeBan: cfg.Security.IPBlacklist.StrikesBeforeBan,
		})
		if err != nil {
			return err
		}
		guard.ApplyOptions(opts)
		return nil
	})

	cfg, _ := util.LoadConfigYAMLOnly(path)
	cfg.Security.IPBlacklist.Enabled = true
	if _, err := store.AdminSave(cfg, nil); err != nil {
		t.Fatal(err)
	}
	if !guard.Enabled() {
		t.Fatal("expected guard enabled after hot reload on save")
	}
}
