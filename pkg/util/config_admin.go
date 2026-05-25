package util

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ConfigSecretsPatch 管理端更新密钥；空字符串或 "***" 表示不修改。
type ConfigSecretsPatch struct {
	JWTSecret         *string `json:"jwt_secret,omitempty"`
	AESSecret         *string `json:"aes_secret,omitempty"`
	DatabasePassword  *string `json:"database_password,omitempty"`
	RedisPassword     *string `json:"redis_password,omitempty"`
	EmailPassword     *string `json:"email_password,omitempty"`
	MetricsBearerToken *string `json:"metrics_bearer_token,omitempty"`
}

// ConfigAdminView 管理端读取的配置视图。
type ConfigAdminView struct {
	ConfigPath    string   `json:"config_path"`
	DotenvPath    string   `json:"dotenv_path,omitempty"`
	File          *Config  `json:"file"`
	Effective     *Config  `json:"effective"`
	EnvOverrides  []string `json:"env_overrides"`
	HotReloadKeys []string `json:"hot_reload_keys"`
	RestartNote   string   `json:"restart_note"`
}

// ConfigSaveResult 保存结果。
type ConfigSaveResult struct {
	Saved         bool              `json:"saved"`
	HotReload     map[string]string `json:"hot_reload,omitempty"`
	RestartNeeded bool              `json:"restart_needed"`
	RestartNote   string            `json:"restart_note"`
}

// HotReloadFunc 热更新回调（无需重启即可生效）。
type HotReloadFunc func(*Config) error

// ConfigStore 配置文件读写与热更新。
type ConfigStore struct {
	path       string
	dotenvPath string
	mu         sync.Mutex
	hotReload  map[string]HotReloadFunc
}

// NewConfigStore 创建配置存储。
func NewConfigStore(configPath, dotenvPath string) *ConfigStore {
	return &ConfigStore{
		path:       configPath,
		dotenvPath: dotenvPath,
		hotReload:  make(map[string]HotReloadFunc),
	}
}

// RegisterHotReload 注册热更新项（如 ip_blacklist）。
func (s *ConfigStore) RegisterHotReload(key string, fn HotReloadFunc) {
	if s == nil || fn == nil {
		return
	}
	s.mu.Lock()
	s.hotReload[key] = fn
	s.mu.Unlock()
}

// AdminGet 返回文件配置、生效配置（含环境变量）及脱敏结果。
func (s *ConfigStore) AdminGet() (*ConfigAdminView, error) {
	if s == nil || s.path == "" {
		return nil, fmt.Errorf("config store not configured")
	}
	fileCfg, err := LoadConfigYAMLOnly(s.path)
	if err != nil {
		return nil, err
	}
	effective, err := LoadConfig(s.path)
	if err != nil {
		return nil, err
	}
	fileCfg.SetDefaults()
	effective.SetDefaults()

	fileCopy := cloneConfig(fileCfg)
	effectiveCopy := cloneConfig(effective)
	maskConfigSecrets(fileCopy)
	maskConfigSecrets(effectiveCopy)

	return &ConfigAdminView{
		ConfigPath:    s.path,
		DotenvPath:    s.dotenvPath,
		File:          fileCopy,
		Effective:     effectiveCopy,
		EnvOverrides:  detectEnvOverrides(),
		HotReloadKeys: []string{"security.ip_blacklist"},
		RestartNote:   "除 IP 黑名单开关与策略外，修改数据库/Redis/JWT/服务端口等通常需重启 ad-platform-main 后完全生效。",
	}, nil
}

// AdminSave 校验并写入 YAML，随后尝试热更新。
func (s *ConfigStore) AdminSave(incoming *Config, secrets *ConfigSecretsPatch) (*ConfigSaveResult, error) {
	if s == nil || s.path == "" {
		return nil, fmt.Errorf("config store not configured")
	}
	if incoming == nil {
		return nil, fmt.Errorf("config is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := LoadConfigYAMLOnly(s.path)
	if err != nil {
		return nil, err
	}
	merged := mergeConfigForSave(current, incoming, secrets)
	merged.SetDefaults()
	if err := merged.ValidateConfig(); err != nil {
		return nil, err
	}
	if err := SaveConfigYAML(s.path, merged); err != nil {
		return nil, err
	}

	result := &ConfigSaveResult{
		Saved:         true,
		HotReload:     make(map[string]string),
		RestartNeeded: true,
		RestartNote:   "配置已写入 config.yaml。除下列热更新项外，请执行 systemctl restart ad-platform-main。",
	}
	effective, err := LoadConfig(s.path)
	if err == nil {
		effective.SetDefaults()
		for key, fn := range s.hotReload {
			if err := fn(effective); err != nil {
				result.HotReload[key] = "error: " + err.Error()
			} else {
				result.HotReload[key] = "ok"
			}
		}
	}
	return result, nil
}

// ApplyHotReload 从磁盘重新加载并对已注册项执行热更新。
func (s *ConfigStore) ApplyHotReload() (map[string]string, error) {
	if s == nil {
		return nil, fmt.Errorf("config store not configured")
	}
	cfg, err := LoadConfig(s.path)
	if err != nil {
		return nil, err
	}
	cfg.SetDefaults()
	out := make(map[string]string)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, fn := range s.hotReload {
		if err := fn(cfg); err != nil {
			out[key] = "error: " + err.Error()
		} else {
			out[key] = "ok"
		}
	}
	return out, nil
}

func cloneConfig(c *Config) *Config {
	if c == nil {
		return nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		cp := *c
		return &cp
	}
	var out Config
	if err := json.Unmarshal(b, &out); err != nil {
		cp := *c
		return &cp
	}
	return &out
}

func maskConfigSecrets(c *Config) {
	if c == nil {
		return
	}
	if c.JWT.Secret != "" {
		c.JWT.Secret = SecretPlaceholder
	}
	if c.Security.AESSecret != "" {
		c.Security.AESSecret = SecretPlaceholder
	}
	if c.Database.Password != "" {
		c.Database.Password = SecretPlaceholder
	}
	for i := range c.Database.Cluster {
		if c.Database.Cluster[i].Password != "" {
			c.Database.Cluster[i].Password = SecretPlaceholder
		}
	}
	if c.Redis.Password != "" {
		c.Redis.Password = SecretPlaceholder
	}
	for i := range c.Redis.Cluster {
		if c.Redis.Cluster[i].Password != "" {
			c.Redis.Cluster[i].Password = SecretPlaceholder
		}
	}
	if c.Email.Password != "" {
		c.Email.Password = SecretPlaceholder
	}
	if c.Metrics.BearerToken != "" {
		c.Metrics.BearerToken = SecretPlaceholder
	}
}

func secretKeepOrReplace(current string, patch *string) string {
	if patch == nil {
		return current
	}
	v := strings.TrimSpace(*patch)
	if v == "" || v == SecretPlaceholder {
		return current
	}
	return v
}

func mergeConfigForSave(current, incoming *Config, secrets *ConfigSecretsPatch) *Config {
	out := cloneConfig(incoming)
	if out == nil {
		out = &Config{}
	}
	if secrets != nil {
		out.JWT.Secret = secretKeepOrReplace(current.JWT.Secret, secrets.JWTSecret)
		out.Security.AESSecret = secretKeepOrReplace(current.Security.AESSecret, secrets.AESSecret)
		out.Database.Password = secretKeepOrReplace(current.Database.Password, secrets.DatabasePassword)
		out.Redis.Password = secretKeepOrReplace(current.Redis.Password, secrets.RedisPassword)
		out.Email.Password = secretKeepOrReplace(current.Email.Password, secrets.EmailPassword)
		out.Metrics.BearerToken = secretKeepOrReplace(current.Metrics.BearerToken, secrets.MetricsBearerToken)
	} else {
		out.JWT.Secret = current.JWT.Secret
		out.Security.AESSecret = current.Security.AESSecret
		out.Database.Password = current.Database.Password
		out.Redis.Password = current.Redis.Password
		out.Email.Password = current.Email.Password
		out.Metrics.BearerToken = current.Metrics.BearerToken
	}
	// 若前端未传密钥字段，保留文件内原值
	if incoming.JWT.Secret == SecretPlaceholder {
		out.JWT.Secret = current.JWT.Secret
	}
	if incoming.Security.AESSecret == SecretPlaceholder {
		out.Security.AESSecret = current.Security.AESSecret
	}
	if incoming.Database.Password == SecretPlaceholder {
		out.Database.Password = current.Database.Password
	}
	if incoming.Redis.Password == SecretPlaceholder {
		out.Redis.Password = current.Redis.Password
	}
	if incoming.Email.Password == SecretPlaceholder {
		out.Email.Password = current.Email.Password
	}
	if incoming.Metrics.BearerToken == SecretPlaceholder {
		out.Metrics.BearerToken = current.Metrics.BearerToken
	}
	return out
}

func detectEnvOverrides() []string {
	var keys []string
	check := func(k string) {
		if GetEnvTrim(k) != "" {
			keys = append(keys, k)
		}
	}
	check(EnvEnvironment)
	check(EnvDBDriver)
	check(EnvDBHost)
	check(EnvDBPort)
	check(EnvDBUser)
	check(EnvDBPassword)
	check(EnvDBName)
	check(EnvJWTSecret)
	check(EnvRedisPassword)
	check(EnvAESSecretKey)
	check(EnvAuthAllowPublicRegister)
	check(EnvEmailEnabled)
	check(EnvEmailHost)
	check(EnvEmailPort)
	check(EnvEmailUsername)
	check(EnvEmailPassword)
	check(EnvEmailFrom)
	check(EnvMetricsBearerToken)
	check(EnvTrustedProxies)
	return keys
}
