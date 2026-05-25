package middleware

import (
	"fmt"
	"net"
	"strings"
)

// NormalizeIP 规范化客户端 IP 字符串。
func NormalizeIP(ip string) (string, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", fmt.Errorf("ip is empty")
	}
	// 去掉方括号（IPv6 URL 形式）
	ip = strings.TrimPrefix(strings.TrimSuffix(ip, "]"), "[")
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid ip address")
	}
	return parsed.String(), nil
}
