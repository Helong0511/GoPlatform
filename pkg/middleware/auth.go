package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"ad-platform/internal/admin"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(userService *admin.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": response.MsgAuthHeaderRequired})
			c.Abort()
			return
		}

		// 解析token获取用户信息
		// 这里简化处理，实际应该使用JWT等认证机制
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": response.MsgBearerFormatRequired})
			c.Abort()
			return
		}

		// 这里应该从token中解析用户ID，这里简化处理
		username := parts[1]
		user, err := userService.GetUserByUsername(username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": response.MsgUserNotFound})
			c.Abort()
			return
		}

		// 检查用户状态
		if !user.Status {
			c.JSON(http.StatusForbidden, gin.H{"error": response.MsgUserDisabled})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Next()
	}
}

// UserContextMiddleware 基于JWT用户ID加载用户详情
func UserContextMiddleware(userService *admin.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := auth.GetUserIDFromContext(c)
		if !exists {
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		user, err := userService.GetUserByID(uint(userID))
		if err != nil {
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		if !user.Status {
			response.Forbidden(c, response.MsgUserDisabled)
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// RBACMiddleware 基于角色的访问控制中间件
func RBACMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取用户信息
		user, exists := c.Get("user")
		if !exists {
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		u := user.(*admin.User)

		// 检查是否为超级管理员
		for _, role := range u.Roles {
			if role.Name == "super_admin" {
				c.Next()
				return
			}
		}

		// 检查用户是否拥有所需权限
		hasPermission := false
		for _, role := range u.Roles {
			for _, perm := range role.Permissions {
				if perm.Name == requiredPermission {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			response.Forbidden(c, response.MsgInsufficientPermissions)
			c.Abort()
			return
		}

		c.Next()
	}
}

// DepartmentMiddleware 部门访问控制中间件
func DepartmentMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取用户信息
		user, exists := c.Get("user")
		if !exists {
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		u := user.(*admin.User)

		// 检查是否为超级管理员
		for _, role := range u.Roles {
			if role.Name == "super_admin" {
				c.Next()
				return
			}
		}

		// 检查用户是否属于某个部门
		if u.DepartmentID == nil {
			response.Forbidden(c, response.MsgDepartmentNotAssigned)
			c.Abort()
			return
		}

		// 将部门信息存储到上下文中
		c.Set("department_id", *u.DepartmentID)
		c.Next()
	}
}
