package util

import (
	"os"
	"strings"
)

// AppVersionString 构建时可由 go build -ldflags "-X ad-platform/pkg/util.AppVersionString=1.2.3" 注入。
var AppVersionString = "dev"

// AppVersion 返回运行版本：优先环境变量 APP_VERSION，其次 ldflags 注入值。
func AppVersion() string {
	if v := strings.TrimSpace(os.Getenv("APP_VERSION")); v != "" {
		return v
	}
	return strings.TrimSpace(AppVersionString)
}
