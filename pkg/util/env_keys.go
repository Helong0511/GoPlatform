package util

import (
	"os"
	"strings"
)

const (
	EnvDBHost        = "DB_HOST"
	EnvDBDriver      = "DB_DRIVER"
	EnvDBPort        = "DB_PORT"
	EnvDBUser        = "DB_USER"
	EnvDBPassword    = "DB_PASSWORD"
	EnvDBName        = "DB_NAME"
	EnvJWTSecret     = "JWT_SECRET"
	EnvRedisPassword = "REDIS_PASSWORD"
	EnvAESSecretKey  = "AES_SECRET_KEY"
	EnvEnvironment   = "ENVIRONMENT"
	// EnvAuthAllowPublicRegister 为 "1"/"true"/"yes"/"on" 时开放公开注册
	EnvAuthAllowPublicRegister = "AUTH_ALLOW_PUBLIC_REGISTER"
	EnvEmailEnabled            = "EMAIL_ENABLED"
	EnvEmailHost               = "EMAIL_HOST"
	EnvEmailPort               = "EMAIL_PORT"
	EnvEmailUsername           = "EMAIL_USERNAME"
	EnvEmailPassword           = "EMAIL_PASSWORD"
	EnvEmailFrom               = "EMAIL_FROM"
)

// GetEnvTrim 返回去除首尾空白后的环境变量值。
func GetEnvTrim(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
