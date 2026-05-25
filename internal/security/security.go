package security

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SecurityManager 安全管理器
// 当前采用 Go 原生实现，并保留 Rust 动态库可选启用状态。
type SecurityManager struct {
	useRustLib   bool
	rustLibPath  string
	aesUtilsPool map[string]*AESUtils
	mu           sync.RWMutex
}

// NewSecurityManager 创建安全管理器
// 若 Rust 动态库文件存在，则标记为可选启用；否则自动回退到 Go 实现。
func NewSecurityManager(libPath string) (*SecurityManager, error) {
	manager := &SecurityManager{
		useRustLib:   false,
		rustLibPath:  strings.TrimSpace(libPath),
		aesUtilsPool: make(map[string]*AESUtils),
	}

	if manager.rustLibPath == "" {
		return manager, nil
	}

	if stat, err := os.Stat(manager.rustLibPath); err == nil && !stat.IsDir() {
		manager.useRustLib = true
	}

	return manager, nil
}

type claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT 生成JWT令牌。
// HTTP API 请优先使用 ad-platform/pkg/auth；此处保留供测试与非 HTTP 场景。
func (s *SecurityManager) GenerateJWT(userID, role, secret string, exp time.Duration) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("jwt secret is required")
	}
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("userID is required")
	}
	if exp <= 0 {
		exp = time.Hour
	}

	now := time.Now().UTC()
	tokenClaims := claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT 验证JWT令牌。
// HTTP API 请优先使用 ad-platform/pkg/auth。
func (s *SecurityManager) ValidateJWT(token, secret string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("jwt token is required")
	}
	if strings.TrimSpace(secret) == "" {
		return false, fmt.Errorf("jwt secret is required")
	}

	tokenClaims := &claims{}
	parsed, err := jwt.ParseWithClaims(token, tokenClaims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return false, err
	}
	return parsed.Valid, nil
}

// Encrypt 加密数据
func (s *SecurityManager) Encrypt(data, key string) (string, error) {
	aesUtils, err := s.getAESUtils(key)
	if err != nil {
		return "", err
	}
	return aesUtils.Encrypt(data)
}

// Decrypt 解密数据
func (s *SecurityManager) Decrypt(ciphertext, key string) (string, error) {
	aesUtils, err := s.getAESUtils(key)
	if err != nil {
		return "", err
	}
	return aesUtils.Decrypt(ciphertext)
}

func (s *SecurityManager) getAESUtils(key string) (*AESUtils, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("aes key is required")
	}

	s.mu.RLock()
	if cached, ok := s.aesUtilsPool[key]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, ok := s.aesUtilsPool[key]; ok {
		return cached, nil
	}
	aesUtils, err := NewAESUtils(key)
	if err != nil {
		return nil, err
	}
	s.aesUtilsPool[key] = aesUtils
	return aesUtils, nil
}

// IsRustLibEnabled 表示 Rust 动态库文件是否可用。
func (s *SecurityManager) IsRustLibEnabled() bool {
	return s.useRustLib
}

// RustLibPath 返回配置的 Rust 动态库路径。
func (s *SecurityManager) RustLibPath() string {
	return s.rustLibPath
}

// Close 关闭安全管理器
func (s *SecurityManager) Close() error {
	// 无需清理资源，直接返回
	return nil
}
