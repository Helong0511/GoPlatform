package util

import "strings"

// IsStrictHardeningEnvironment 预发/生产等需强制 SSL、密钥与访问控制的环境。
func IsStrictHardeningEnvironment(env string) bool {
	e := strings.ToLower(strings.TrimSpace(env))
	switch e {
	case "production", "prod", "staging", "uat", "preprod", "pre-prod":
		return true
	default:
		return false
	}
}

func effectiveEnvOrConfig(envKey, configValue string) string {
	if v := GetEnvTrim(envKey); v != "" {
		return v
	}
	return strings.TrimSpace(configValue)
}

// looksLikePlaceholder 检测未替换的模板占位符。
func looksLikePlaceholder(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return true
	}
	return strings.Contains(s, "replace_me") || strings.Contains(s, "changeme") || s == "your_secret" || s == "todo"
}
