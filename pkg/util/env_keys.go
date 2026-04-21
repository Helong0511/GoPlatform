package util

import (
	"os"
	"strings"
)

const (
	EnvDBHost        = "DB_HOST"
	EnvDBPort        = "DB_PORT"
	EnvDBUser        = "DB_USER"
	EnvDBPassword    = "DB_PASSWORD"
	EnvDBName        = "DB_NAME"
	EnvJWTSecret     = "JWT_SECRET"
	EnvRedisPassword = "REDIS_PASSWORD"
	EnvAESSecretKey  = "AES_SECRET_KEY"
)

// GetEnvTrim 返回去除首尾空白后的环境变量值。
func GetEnvTrim(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
