package user

import (
	"errors"
	"strconv"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService userService
}

type userService interface {
	ListUsers(page, pageSize int) ([]admin.User, int64, error)
	GetUserByID(id uint) (*admin.User, error)
	CreateUser(user *admin.User) error
	UpdateUser(user *admin.User) error
	DeleteUser(id uint) error
	EnableUser(id, operatorID uint, ip string) error
	DisableUser(id, operatorID uint, ip string) error
	GetUserStatusLogs(userID uint, page, pageSize int) ([]admin.UserStatusLog, int64, error)
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *admin.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// NewUserHandlerWithService 用于测试和自定义实现注入。
func NewUserHandlerWithService(userService userService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// SetupRoutes 设置用户相关路由
func (h *UserHandler) SetupRoutes(protected *gin.RouterGroup) {
	users := protected.Group("/users")
	{
		users.GET("", middleware.RBACMiddleware("user:read"), h.listUsers)
		users.GET("/:id", middleware.RBACMiddleware("user:read"), h.getUser)
		users.POST("", middleware.RBACMiddleware("user:create"), h.createUser)
		users.PUT("/:id", middleware.RBACMiddleware("user:update"), h.updateUser)
		users.DELETE("/:id", middleware.RBACMiddleware("user:delete"), h.deleteUser)
		users.PUT("/:id/enable", middleware.RBACMiddleware("user:update"), h.enableUser)
		users.PUT("/:id/disable", middleware.RBACMiddleware("user:update"), h.disableUser)
		users.GET("/:id/status-logs", middleware.RBACMiddleware("user:read"), h.getUserStatusLogs)
	}
}

// listUsers 列出用户
func (h *UserHandler) listUsers(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	users, total, err := h.userService.ListUsers(page, pageSize)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveUsers)
		return
	}

	response.SuccessPaginated(c, response.MsgUsersRetrievedSuccessfully, users, total, page, pageSize)
}

// getUser 获取用户
func (h *UserHandler) getUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		if errors.Is(err, common.ErrUserNotFound) {
			response.NotFound(c, response.MsgUserNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedRetrieveUser)
		return
	}

	response.SuccessData(c, response.MsgUserRetrievedSuccessfully, user)
}

// createUser 创建用户
func (h *UserHandler) createUser(c *gin.Context) {
	var user admin.User
	if err := c.ShouldBindJSON(&user); err != nil {
		response.BadRequest(c, response.MsgInvalidUserData)
		return
	}

	if err := h.userService.CreateUser(&user); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateUser)
		return
	}

	response.Created(c, response.MsgUserCreatedSuccessfully, user)
}

// updateUser 更新用户
func (h *UserHandler) updateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	var user admin.User
	if err := c.ShouldBindJSON(&user); err != nil {
		response.BadRequest(c, response.MsgInvalidUserData)
		return
	}

	user.ID = uint(id)
	if err := h.userService.UpdateUser(&user); err != nil {
		response.InternalServerError(c, response.MsgFailedUpdateUser)
		return
	}

	response.SuccessData(c, response.MsgUserUpdatedSuccessfully, user)
}

// deleteUser 删除用户
func (h *UserHandler) deleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		response.InternalServerError(c, response.MsgFailedDeleteUser)
		return
	}

	response.Success(c, response.MsgUserDeletedSuccessfully)
}

// enableUser 启用用户
func (h *UserHandler) enableUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	// 从JWT上下文中获取操作者ID
	userIDStr, exists := auth.GetUserIDFromContext(c)
	if !exists {
		response.Unauthorized(c, response.MsgUnauthorized)
		return
	}
	operatorID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.Unauthorized(c, response.MsgInvalidOperatorIdentity)
		return
	}
	clientIP := c.ClientIP()

	if err := h.userService.EnableUser(uint(id), uint(operatorID), clientIP); err != nil {
		response.InternalServerError(c, response.MsgFailedEnableUser)
		return
	}

	response.Success(c, response.MsgUserEnabledSuccessfully)
}

// disableUser 禁用用户
func (h *UserHandler) disableUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	// 从JWT上下文中获取操作者ID
	userIDStr, exists := auth.GetUserIDFromContext(c)
	if !exists {
		response.Unauthorized(c, response.MsgUnauthorized)
		return
	}
	operatorID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.Unauthorized(c, response.MsgInvalidOperatorIdentity)
		return
	}
	clientIP := c.ClientIP()

	if err := h.userService.DisableUser(uint(id), uint(operatorID), clientIP); err != nil {
		response.InternalServerError(c, response.MsgFailedDisableUser)
		return
	}

	response.Success(c, response.MsgUserDisabledSuccessfully)
}

// getUserStatusLogs 获取用户状态变更日志
func (h *UserHandler) getUserStatusLogs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	logs, total, err := h.userService.GetUserStatusLogs(uint(id), page, pageSize)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveUserStatusLogs)
		return
	}

	response.SuccessPaginated(c, response.MsgUserStatusLogsRetrievedSuccessfully, logs, total, page, pageSize)
}
