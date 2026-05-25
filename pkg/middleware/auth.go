package middleware

import (
	"errors"
	"strconv"
	"strings"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserContextMiddleware 基于JWT用户ID加载用户详情
func UserContextMiddleware(userService *admin.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := auth.GetUserIDFromContext(c)
		if !exists {
			response.SetErrorReason(c, response.ReasonUserNotAuthenticated)
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			response.SetErrorReason(c, response.ReasonUserNotAuthenticated)
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		user, err := userService.GetUserByID(uint(userID))
		if err != nil {
			response.SetErrorReason(c, response.ReasonUserNotAuthenticated)
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		if !user.Status {
			response.SetErrorReason(c, response.ReasonUserLocked)
			response.Forbidden(c, response.MsgUserLockedCannotLogin)
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
			response.SetErrorReason(c, response.ReasonUserNotAuthenticated)
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		u, ok := user.(*admin.User)
		if !ok {
			response.SetErrorReason(c, response.ReasonInvalidUserPrincipal)
			response.InternalServerError(c, response.MsgInvalidUserPrincipal)
			c.Abort()
			return
		}

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
			response.SetErrorReason(c, response.ReasonInsufficientPermission)
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
			response.SetErrorReason(c, response.ReasonUserNotAuthenticated)
			response.Unauthorized(c, response.MsgUserNotAuthenticated)
			c.Abort()
			return
		}

		u, ok := user.(*admin.User)
		if !ok {
			response.SetErrorReason(c, response.ReasonInvalidUserPrincipal)
			response.InternalServerError(c, response.MsgInvalidUserPrincipal)
			c.Abort()
			return
		}

		// 检查是否为超级管理员
		for _, role := range u.Roles {
			if role.Name == "super_admin" {
				c.Next()
				return
			}
		}

		// 检查用户是否属于某个部门
		if u.DepartmentID == nil {
			response.SetErrorReason(c, response.ReasonDepartmentNotAssigned)
			response.Forbidden(c, response.MsgDepartmentNotAssigned)
			c.Abort()
			return
		}

		// 将部门信息存储到上下文中
		c.Set("department_id", *u.DepartmentID)
		c.Next()
	}
}

// APIKeyMiddleware 通过 X-API-Key 校验下游访问身份
func APIKeyMiddleware(userService *admin.UserService, requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if apiKey == "" {
			response.SetErrorReason(c, response.ReasonMissingAPIKey)
			response.Unauthorized(c, response.MsgAPIKeyRequired)
			c.Abort()
			return
		}

		user, err := userService.ValidateAPIKey(apiKey, requiredScope, c.ClientIP())
		if err != nil {
			if errors.Is(err, common.ErrForbidden) {
				response.SetErrorReason(c, response.ReasonAPIKeyScopeDenied)
				response.Forbidden(c, response.MsgAPIKeyScopeDenied)
				c.Abort()
				return
			}
			response.SetErrorReason(c, response.ReasonInvalidAPIKey)
			response.Unauthorized(c, response.MsgInvalidAPIKey)
			c.Abort()
			return
		}
		if !user.Status {
			response.SetErrorReason(c, response.ReasonUserLocked)
			response.Forbidden(c, response.MsgUserLockedCannotLogin)
			c.Abort()
			return
		}

		c.Set("api_user_id", user.ID)
		c.Set("api_user", user)
		c.Next()
	}
}
