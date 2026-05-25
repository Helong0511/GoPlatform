package util

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// IsProductionEnvironment 判断是否按生产环境约束处理（密钥强度、CORS 等）。
func IsProductionEnvironment(env string) bool {
	e := strings.ToLower(strings.TrimSpace(env))
	return e == "production" || e == "prod"
}

// ValidateProductionSecrets 校验生产环境所需的 JWT / AES 密钥强度（需在 ResolveSecret 之后调用）。
func ValidateProductionSecrets(jwtSecret, aesSecretBase64 string) error {
	jwtSecret = strings.TrimSpace(jwtSecret)
	if len(jwtSecret) < 32 {
		return fmt.Errorf("jwt secret must be at least 32 characters in production")
	}
	if jwtSecretLooksWeak(jwtSecret) {
		return fmt.Errorf("jwt secret appears too weak for production (low entropy or repeated characters)")
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(aesSecretBase64))
	if err != nil {
		return fmt.Errorf("aes secret must be valid base64 in production: %w", err)
	}
	n := len(key)
	if n != 16 && n != 24 && n != 32 {
		return fmt.Errorf("aes secret must decode to 16, 24, or 32 bytes in production, got %d", n)
	}
	return nil
}

func jwtSecretLooksWeak(secret string) bool {
	if len(secret) < 8 {
		return true
	}
	allSame := true
	for i := 1; i < len(secret); i++ {
		if secret[i] != secret[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return true
	}
	// 常见占位串
	lower := strings.ToLower(secret)
	for _, placeholder := range []string{"changeme", "password", "secret", "jwt_secret", "your_jwt"} {
		if strings.Contains(lower, placeholder) {
			return true
		}
	}
	return false
}
