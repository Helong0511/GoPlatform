package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"ad-platform/pkg/logger"
	"ad-platform/pkg/redis"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	redisIPBanKeyPrefix   = "goplatform:ipban:"
	redisIPAllowKeyPrefix = "goplatform:ipallow:"
)

// IPGuardOptions IP 自动黑名单配置。
type IPGuardOptions struct {
	Enabled                    bool
	BanDuration                time.Duration // 0 表示永久封禁
	StrikeWindow               time.Duration
	StrikesBeforeBan           int
	BlockCrawlerUserAgent      bool
	CrawlerStrikeWeight        int
	StaticBlockedIPs           []string
	StaticWhitelistIPs         []string
}

// BannedIPEntry 封禁记录（管理端查询）。
type BannedIPEntry struct {
	IP        string     `json:"ip"`
	Reason    string     `json:"reason"`
	Source    string     `json:"source"`
	BannedAt  time.Time  `json:"banned_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Permanent bool       `json:"permanent"`
	Static    bool       `json:"static"`
}

// WhitelistIPEntry 白名单记录（管理端查询）。
type WhitelistIPEntry struct {
	IP       string    `json:"ip"`
	Remark   string    `json:"remark"`
	Source   string    `json:"source"`
	AddedAt  time.Time `json:"added_at"`
	Static   bool      `json:"static"`
}

// IPGuard 根据敏感路径访问、爬虫 UA、认证失败等自动封禁 IP。
type IPGuard struct {
	cfg   IPGuardOptions
	redis *redis.RedisClient

	mu     sync.RWMutex
	banned map[string]banRecord
	strikes    map[string]*ipStrikeBucket
	static     map[string]struct{}
	whitelist  map[string]whitelistRecord
	staticAllow map[string]struct{}
}

type banRecord struct {
	reason    string
	source    string
	bannedAt  time.Time
	expiresAt time.Time // IsZero 表示永久
}

type whitelistRecord struct {
	remark  string
	source  string
	addedAt time.Time
}

type ipStrikeBucket struct {
	events []strikeEvent
}

type strikeEvent struct {
	at     time.Time
	reason string
	weight int
}

// NewIPGuard 创建 IP 防护；redis 可为 nil 或 mock（仅内存封禁）。
func NewIPGuard(cfg IPGuardOptions, redisClient *redis.RedisClient) *IPGuard {
	g := &IPGuard{
		cfg:     cfg,
		redis:   redisClient,
		banned:  make(map[string]banRecord),
		strikes:     make(map[string]*ipStrikeBucket),
		static:      make(map[string]struct{}),
		whitelist:   make(map[string]whitelistRecord),
		staticAllow: make(map[string]struct{}),
	}
	now := time.Now().UTC()
	for _, raw := range cfg.StaticBlockedIPs {
		ip, err := NormalizeIP(raw)
		if err != nil {
			continue
		}
		g.static[ip] = struct{}{}
		g.mu.Lock()
		g.banned[ip] = banRecord{
			reason:   "static_config",
			source:   "static_config",
			bannedAt: now,
		}
		g.mu.Unlock()
	}
	for _, raw := range cfg.StaticWhitelistIPs {
		ip, err := NormalizeIP(raw)
		if err != nil {
			continue
		}
		g.staticAllow[ip] = struct{}{}
		g.mu.Lock()
		g.whitelist[ip] = whitelistRecord{
			remark:  "static_config",
			source:  "static_config",
			addedAt: now,
		}
		g.mu.Unlock()
	}
	return g
}

// Enabled 是否启用自动黑名单。
func (g *IPGuard) Enabled() bool {
	return g != nil && g.cfg.Enabled
}

// ApplyOptions 热更新运行时策略（不重建静态黑白名单）。
func (g *IPGuard) ApplyOptions(opts IPGuardOptions) {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.cfg.Enabled = opts.Enabled
	g.cfg.BanDuration = opts.BanDuration
	g.cfg.StrikeWindow = opts.StrikeWindow
	g.cfg.StrikesBeforeBan = opts.StrikesBeforeBan
	g.cfg.BlockCrawlerUserAgent = opts.BlockCrawlerUserAgent
	g.cfg.CrawlerStrikeWeight = opts.CrawlerStrikeWeight
	g.mu.Unlock()
}

// Middleware 前置拦截已封禁 IP，请求结束后累计 strike 并可能自动封禁。
func (g *IPGuard) Middleware() gin.HandlerFunc {
	if g == nil || !g.cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if g.IsWhitelisted(ip) {
			c.Next()
			return
		}
		if banned, _ := g.isBanned(ip); banned {
			response.SetErrorReason(c, response.ReasonIPBlacklisted)
			response.Forbidden(c, response.MsgIPBlacklisted)
			c.Abort()
			return
		}
		c.Next()
		g.evaluateAfterRequest(c, ip)
	}
}

func (g *IPGuard) evaluateAfterRequest(c *gin.Context, ip string) {
	if ip == "" || g.IsWhitelisted(ip) {
		return
	}
	path := c.Request.URL.Path
	method := c.Request.Method
	status := c.Writer.Status()
	ua := c.GetHeader("User-Agent")

	if g.cfg.BlockCrawlerUserAgent && isSensitivePath(path) && isCrawlerUserAgent(ua) {
		weight := g.cfg.CrawlerStrikeWeight
		if weight <= 0 {
			weight = 2
		}
		g.addStrike(ip, "crawler_sensitive", weight)
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		if isSensitivePath(path) || isAuthSensitivePath(path) {
			g.addStrike(ip, strikeReasonFromContext(c), 1)
		}
	case status == http.StatusNotFound:
		if isSensitivePath(path) || looksLikeProbePath(path) {
			g.addStrike(ip, "sensitive_probe", 1)
		}
	case status == http.StatusTooManyRequests:
		if isSensitivePath(path) || isAuthSensitivePath(path) {
			g.addStrike(ip, "rate_limited_sensitive", 1)
		}
	}

	_ = method
}

func strikeReasonFromContext(c *gin.Context) string {
	if v, ok := c.Get(response.ContextErrorReasonKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "sensitive_denied"
}

func (g *IPGuard) addStrike(ip, reason string, weight int) {
	if weight <= 0 {
		weight = 1
	}
	now := time.Now().UTC()
	g.mu.Lock()
	bucket, ok := g.strikes[ip]
	if !ok {
		bucket = &ipStrikeBucket{}
		g.strikes[ip] = bucket
	}
	bucket.events = append(bucket.events, strikeEvent{at: now, reason: reason, weight: weight})
	total := bucket.pruneAndSum(g.cfg.StrikeWindow, now)
	threshold := g.cfg.StrikesBeforeBan
	g.mu.Unlock()

	if threshold > 0 && total >= threshold {
		_ = g.BanWithSource(ip, reason+"_auto", "auto")
	}
}

func (b *ipStrikeBucket) pruneAndSum(window time.Duration, now time.Time) int {
	if window <= 0 {
		window = 10 * time.Minute
	}
	cutoff := now.Add(-window)
	sum := 0
	kept := b.events[:0]
	for _, e := range b.events {
		if e.at.After(cutoff) {
			kept = append(kept, e)
			sum += e.weight
		}
	}
	b.events = kept
	return sum
}

// IsWhitelisted 是否在白名单（白名单 IP 不封禁、不计 strike）。
func (g *IPGuard) IsWhitelisted(ip string) bool {
	if g == nil || ip == "" {
		return false
	}
	if norm, err := NormalizeIP(ip); err == nil {
		ip = norm
	}
	if _, ok := g.staticAllow[ip]; ok {
		return true
	}
	g.mu.RLock()
	_, ok := g.whitelist[ip]
	g.mu.RUnlock()
	if ok {
		return true
	}
	if g.redis != nil {
		val, err := g.redis.Get(redisIPAllowKey(ip))
		if err == nil && val != "" {
			return true
		}
	}
	return false
}

// Ban 封禁 IP（手动或自动）；source 为空时视为 manual。
func (g *IPGuard) Ban(ip, reason string) error {
	return g.BanWithSource(ip, reason, "manual")
}

// BanWithSource 封禁 IP 并记录来源（manual / auto 等）。
func (g *IPGuard) BanWithSource(ip, reason, source string) error {
	if g == nil || ip == "" {
		return fmt.Errorf("invalid guard or ip")
	}
	norm, err := NormalizeIP(ip)
	if err != nil {
		return err
	}
	ip = norm
	if g.IsWhitelisted(ip) {
		return fmt.Errorf("ip is whitelisted")
	}
	if source == "" {
		source = "manual"
	}
	if reason == "" {
		reason = source
	}
	now := time.Now().UTC()
	var expiresAt time.Time
	var redisTTL time.Duration
	if g.cfg.BanDuration > 0 {
		expiresAt = now.Add(g.cfg.BanDuration)
		redisTTL = g.cfg.BanDuration
	}

	g.mu.Lock()
	g.banned[ip] = banRecord{reason: reason, source: source, bannedAt: now, expiresAt: expiresAt}
	delete(g.strikes, ip)
	g.mu.Unlock()

	if g.redis != nil && redisTTL > 0 {
		_ = g.redis.Set(redisIPBanKey(ip), reason, redisTTL)
	} else if g.redis != nil && g.cfg.BanDuration == 0 {
		_ = g.redis.Set(redisIPBanKey(ip), reason, 10*365*24*time.Hour)
	}

	logger.S().Warn("ip blacklisted",
		"ip", ip,
		"reason", reason,
		"source", source,
		"ban_duration", g.cfg.BanDuration.String(),
	)
	return nil
}

// Unban 解除封禁（配置文件中的静态黑名单不可解封）。
func (g *IPGuard) Unban(ip string) bool {
	if g == nil || ip == "" {
		return false
	}
	norm, err := NormalizeIP(ip)
	if err != nil {
		return false
	}
	ip = norm
	if _, isStatic := g.static[ip]; isStatic {
		return false
	}
	g.mu.Lock()
	_, had := g.banned[ip]
	delete(g.banned, ip)
	delete(g.strikes, ip)
	g.mu.Unlock()
	if g.redis != nil {
		_ = g.redis.Delete(redisIPBanKey(ip))
	}
	return had
}

// AddWhitelist 加入白名单；若已在黑名单则先解封。
func (g *IPGuard) AddWhitelist(ip, remark string) error {
	if g == nil {
		return fmt.Errorf("ip guard not configured")
	}
	norm, err := NormalizeIP(ip)
	if err != nil {
		return err
	}
	ip = norm
	if remark == "" {
		remark = "manual"
	}
	_ = g.Unban(ip)
	now := time.Now().UTC()
	g.mu.Lock()
	g.whitelist[ip] = whitelistRecord{remark: remark, source: "manual", addedAt: now}
	g.mu.Unlock()
	if g.redis != nil {
		_ = g.redis.Set(redisIPAllowKey(ip), remark, 0)
	}
	return nil
}

// RemoveWhitelist 移除白名单（静态配置不可删）。
func (g *IPGuard) RemoveWhitelist(ip string) bool {
	if g == nil || ip == "" {
		return false
	}
	norm, err := NormalizeIP(ip)
	if err != nil {
		return false
	}
	ip = norm
	if _, isStatic := g.staticAllow[ip]; isStatic {
		return false
	}
	g.mu.Lock()
	_, had := g.whitelist[ip]
	delete(g.whitelist, ip)
	g.mu.Unlock()
	if g.redis != nil {
		_ = g.redis.Delete(redisIPAllowKey(ip))
	}
	return had
}

// ListWhitelist 返回白名单列表。
func (g *IPGuard) ListWhitelist() []WhitelistIPEntry {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]WhitelistIPEntry, 0, len(g.whitelist))
	for ip, rec := range g.whitelist {
		_, isStatic := g.staticAllow[ip]
		out = append(out, WhitelistIPEntry{
			IP:      ip,
			Remark:  rec.remark,
			Source:  rec.source,
			AddedAt: rec.addedAt,
			Static:  isStatic,
		})
	}
	return out
}

// IPGuardStatus 管理端展示的防护状态。
type IPGuardStatus struct {
	Enabled bool `json:"enabled"`
}

// Status 返回防护开关状态。
func (g *IPGuard) Status() IPGuardStatus {
	return IPGuardStatus{Enabled: g != nil && g.cfg.Enabled}
}

// ListBanned 返回当前内存中的封禁列表（不含已过期）。
func (g *IPGuard) ListBanned() []BannedIPEntry {
	if g == nil {
		return nil
	}
	now := time.Now().UTC()
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]BannedIPEntry, 0, len(g.banned))
	for ip, rec := range g.banned {
		if !rec.expiresAt.IsZero() && now.After(rec.expiresAt) {
			continue
		}
		_, isStatic := g.static[ip]
		entry := BannedIPEntry{
			IP:       ip,
			Reason:   rec.reason,
			Source:   rec.source,
			BannedAt: rec.bannedAt,
			Static:   isStatic,
		}
		if rec.expiresAt.IsZero() {
			entry.Permanent = true
		} else {
			t := rec.expiresAt
			entry.ExpiresAt = &t
		}
		out = append(out, entry)
	}
	return out
}

func (g *IPGuard) isBanned(ip string) (bool, string) {
	if g == nil || ip == "" {
		return false, ""
	}
	if norm, err := NormalizeIP(ip); err == nil {
		ip = norm
	}
	if g.IsWhitelisted(ip) {
		return false, ""
	}
	if _, ok := g.static[ip]; ok {
		return true, "static_config"
	}

	now := time.Now().UTC()
	g.mu.RLock()
	rec, ok := g.banned[ip]
	g.mu.RUnlock()
	if ok {
		if rec.expiresAt.IsZero() || now.Before(rec.expiresAt) {
			return true, rec.reason
		}
		g.mu.Lock()
		delete(g.banned, ip)
		g.mu.Unlock()
	}

	if g.redis != nil {
		val, err := g.redis.Get(redisIPBanKey(ip))
		if err == nil && val != "" {
			g.mu.Lock()
			g.banned[ip] = banRecord{reason: val, bannedAt: now}
			g.mu.Unlock()
			return true, val
		}
	}
	return false, ""
}

func redisIPBanKey(ip string) string {
	return redisIPBanKeyPrefix + ip
}

func redisIPAllowKey(ip string) string {
	return redisIPAllowKeyPrefix + ip
}

func isSensitivePath(path string) bool {
	p := strings.TrimSuffix(path, "/")
	if p == "" {
		return false
	}
	return strings.HasPrefix(p, "/api/admin")
}

func isAuthSensitivePath(path string) bool {
	p := strings.TrimSuffix(path, "/")
	switch p {
	case "/api/login", "/api/register":
		return true
	default:
		return strings.HasPrefix(p, "/api/report")
	}
}

func looksLikeProbePath(path string) bool {
	lower := strings.ToLower(path)
	probes := []string{
		"/.env", "/wp-admin", "/wp-login", "/phpmyadmin", "/admin.php",
		"/.git", "/config", "/swagger", "/actuator", "/server-status",
	}
	for _, p := range probes {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isCrawlerUserAgent(ua string) bool {
	if ua == "" {
		return false
	}
	lower := strings.ToLower(ua)
	patterns := []string{
		"bot", "spider", "crawl", "scrapy", "headless",
		"python-requests", "curl/", "wget/", "go-http-client",
		"java/", "okhttp", "libwww", "perl", "ruby", "php/",
		"semrush", "ahrefs", "petalbot", "bytespider",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
