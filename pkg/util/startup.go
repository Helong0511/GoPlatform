package util

import (
	"fmt"
	"strings"
)

const (
	SecretSourceEnv    = "env"
	SecretSourceConfig = "config"
)

// ResolveSecret 统一解析启动密钥来源：
// 1. 环境变量优先；2. 配置文件兜底；3. 都为空返回错误。
func ResolveSecret(envKey, configValue string) (value string, source string, err error) {
	if v := GetEnvTrim(envKey); v != "" {
		return v, SecretSourceEnv, nil
	}
	if v := strings.TrimSpace(configValue); v != "" {
		return v, SecretSourceConfig, nil
	}
	return "", "", fmt.Errorf("secret %s is required but missing in env and config", envKey)
}
