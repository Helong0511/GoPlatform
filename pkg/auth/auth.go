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
	UserID   string   `json:"user_id"`
	Role     string   `json:"role"`            // 兼容旧客户端：首个角色名，与 Roles[0] 一致
	Roles    []string `json:"roles,omitempty"` // 全部角色名（去重、保持顺序）
	Timezone string   `json:"timezone"`
	jwt.RegisteredClaims
}

// AuthMiddleware 认证中间件管理器
type AuthMiddleware struct {
	jwtSecret string
	issuer    string
	audience  string
}

// NewAuthMiddleware 创建认证中间件管理器；issuer/audience 为空时不校验对应声明。
func NewAuthMiddleware(jwtSecret, issuer, audience string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: strings.TrimSpace(jwtSecret),
		issuer:    strings.TrimSpace(issuer),
		audience:  strings.TrimSpace(audience),
	}
}

// GenerateJWT 生成 JWT；roleNames 为全部角色名（首个写入 role 声明便于旧解析逻辑）。
func (a *AuthMiddleware) GenerateJWT(userID, timezone string, roleNames []string, exp time.Duration) (string, error) {
	primary := ""
	if len(roleNames) > 0 {
		primary = roleNames[0]
	}
	now := time.Now().UTC()
	rc := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	if a.issuer != "" {
		rc.Issuer = a.issuer
	}
	if a.audience != "" {
		rc.Audience = jwt.ClaimStrings{a.audience}
	}

	claims := Claims{
		UserID:           userID,
		Role:             primary,
		Roles:            roleNames,
		Timezone:         timezone,
		RegisteredClaims: rc,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

func (a *AuthMiddleware) parseOptions() []jwt.ParserOption {
	opts := make([]jwt.ParserOption, 0, 2)
	if a.issuer != "" {
		opts = append(opts, jwt.WithIssuer(a.issuer))
	}
	if a.audience != "" {
		opts = append(opts, jwt.WithAudience(a.audience))
	}
	return opts
}

// ValidateJWT 验证JWT令牌（含 iss/aud 与 HMAC 算法白名单）
func (a *AuthMiddleware) ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.jwtSecret), nil
	}, a.parseOptions()...)

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
			response.SetErrorReason(c, response.ReasonMissingAuthHeader)
			response.Unauthorized(c, response.MsgAuthHeaderRequired)
			c.Abort()
			return
		}

		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			response.SetErrorReason(c, response.ReasonInvalidBearerFormat)
			response.Unauthorized(c, response.MsgBearerFormatRequired)
			c.Abort()
			return
		}
		tokenString := strings.TrimSpace(authHeader[len("Bearer "):])
		if tokenString == "" {
			response.SetErrorReason(c, response.ReasonMissingBearerToken)
			response.Unauthorized(c, response.MsgBearerTokenRequired)
			c.Abort()
			return
		}

		claims, err := a.ValidateJWT(tokenString)
		if err != nil {
			response.SetErrorReason(c, response.ReasonInvalidOrExpiredToken)
			response.Unauthorized(c, response.MsgInvalidOrExpiredToken)
			c.Abort()
			return
		}

		roleNames := append([]string{}, claims.Roles...)
		if len(roleNames) == 0 && strings.TrimSpace(claims.Role) != "" {
			roleNames = []string{strings.TrimSpace(claims.Role)}
		}
		primary := strings.TrimSpace(claims.Role)
		if primary == "" && len(roleNames) > 0 {
			primary = roleNames[0]
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", primary)
		c.Set("roles", roleNames)
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

// GetRolesFromContext 从上下文中获取全部角色名（可能为空切片）。
func GetRolesFromContext(c *gin.Context) ([]string, bool) {
	v, exists := c.Get("roles")
	if !exists {
		if r, ok := GetRoleFromContext(c); ok && r != "" {
			return []string{r}, true
		}
		return nil, false
	}
	switch x := v.(type) {
	case []string:
		return x, true
	default:
		return nil, false
	}
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
