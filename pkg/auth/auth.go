// Package auth 统一认证包
// 提供标准化的JWT认证中间件和认证相关功能
package auth

import (
	"errors"
	"strings"
	"time"

	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT声明结构
type Claims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Timezone string `json:"timezone"`
	jwt.RegisteredClaims
}

// AuthMiddleware 认证中间件管理器
type AuthMiddleware struct {
	jwtSecret string
}

// NewAuthMiddleware 创建认证中间件管理器
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// GenerateJWT 生成JWT令牌
func (a *AuthMiddleware) GenerateJWT(userID, role, timezone string, exp time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Role:     role,
		Timezone: timezone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// ValidateJWT 验证JWT令牌
func (a *AuthMiddleware) ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}

// AuthMiddleware 认证中间件
func (a *AuthMiddleware) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Unauthorized(c, response.MsgAuthHeaderRequired)
			c.Abort()
			return
		}

		// 严格要求 Bearer 前缀，避免兼容模式带来的歧义
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			response.Unauthorized(c, response.MsgBearerFormatRequired)
			c.Abort()
			return
		}
		tokenString := strings.TrimSpace(authHeader[len("Bearer "):])
		if tokenString == "" {
			response.Unauthorized(c, response.MsgBearerTokenRequired)
			c.Abort()
			return
		}

		// 解析token
		claims, err := a.ValidateJWT(tokenString)
		if err != nil {
			response.Unauthorized(c, response.MsgInvalidOrExpiredToken)
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("timezone", claims.Timezone)

		c.Next()
	}
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		return "", false
	}
	return userIDStr, true
}

// GetRoleFromContext 从上下文中获取角色
func GetRoleFromContext(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	roleStr, ok := role.(string)
	if !ok {
		return "", false
	}
	return roleStr, true
}

// GetTimezoneFromContext 从上下文中获取时区
func GetTimezoneFromContext(c *gin.Context) (string, bool) {
	timezone, exists := c.Get("timezone")
	if !exists {
		return "", false
	}
	timezoneStr, ok := timezone.(string)
	if !ok {
		return "", false
	}
	return timezoneStr, true
}
