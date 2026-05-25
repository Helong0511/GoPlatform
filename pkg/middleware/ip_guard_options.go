package middleware

import (
	"fmt"
	"time"
)

// IPBlacklistYAML 与 config.yaml security.ip_blacklist 字段对应（避免 util ↔ middleware 循环依赖）。
type IPBlacklistYAML struct {
	Enabled               bool
	BanDuration           string
	StrikeWindow          string
	StrikesBeforeBan      int
	BlockCrawlerUserAgent bool
	CrawlerStrikeWeight   int
	StaticIPs             []string
	StaticWhitelistIPs    []string
}

// ParseIPGuardOptions 解析 YAML 配置为运行时选项。
func ParseIPGuardOptions(y IPBlacklistYAML) (IPGuardOptions, error) {
	opts := IPGuardOptions{
		Enabled:               y.Enabled,
		BlockCrawlerUserAgent: y.BlockCrawlerUserAgent,
		CrawlerStrikeWeight:   y.CrawlerStrikeWeight,
		StaticBlockedIPs:      append([]string(nil), y.StaticIPs...),
		StaticWhitelistIPs:    append([]string(nil), y.StaticWhitelistIPs...),
		StrikesBeforeBan:      y.StrikesBeforeBan,
	}
	if y.BanDuration != "" {
		d, err := time.ParseDuration(y.BanDuration)
		if err != nil {
			return opts, fmt.Errorf("ban_duration: %w", err)
		}
		opts.BanDuration = d
	}
	if y.StrikeWindow != "" {
		d, err := time.ParseDuration(y.StrikeWindow)
		if err != nil {
			return opts, fmt.Errorf("strike_window: %w", err)
		}
		opts.StrikeWindow = d
	}
	return opts, nil
}
