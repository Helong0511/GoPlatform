package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SecretPlaceholder 管理端展示/回传时表示“已配置且不修改”。
const SecretPlaceholder = "***"

// LoadConfigYAMLOnly 仅从 YAML 文件加载（不合并环境变量）。
func LoadConfigYAMLOnly(filePath string) (*Config, error) {
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
	return &config, nil
}

// SaveConfigYAML 原子写入 YAML 配置文件。
func SaveConfigYAML(filePath string, config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out := *config
	out.environmentFromFile = ""
	data, err := yaml.Marshal(&out)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filePath)
}
