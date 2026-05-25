package user

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/notify"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService userService
	emailSender notify.EmailSender
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

type registrationService interface {
	CountPendingRegistrations() (int64, error)
	ListPendingRegistrations(page, pageSize int) ([]admin.UserRegistration, int64, error)
	ReviewRegistration(registrationID, operatorID uint, approved bool, reason string) (*admin.RegistrationReviewResult, error)
	ListRegistrationAudits(registrationID uint) ([]admin.UserRegistrationAudit, error)
}

type apiKeyService interface {
	ListUserAPIKeys(userID uint) ([]admin.UserAPIKey, error)
	DisableUserAPIKey(userID, keyID uint) error
	RotateUserAPIKey(userID uint, name, scope string, expiresAt *time.Time) (string, error)
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *admin.UserService, emailSender ...notify.EmailSender) *UserHandler {
	var sender notify.EmailSender
	if len(emailSender) > 0 {
		sender = emailSender[0]
	}
	return &UserHandler{
		userService: userService,
		emailSender: sender,
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
		users.GET("/registrations/pending-count", middleware.RBACMiddleware("user:read"), h.getPendingRegistrationCount)
		users.GET("/registrations", middleware.RBACMiddleware("user:read"), h.listPendingRegistrations)
		users.POST("/registrations/:id/review", middleware.RBACMiddleware("user:update"), h.reviewRegistration)
		users.GET("/registrations/:id/audits", middleware.RBACMiddleware("user:read"), h.listRegistrationAudits)
		users.GET("/:id/api-keys", middleware.RBACMiddleware("user:read"), h.listUserAPIKeys)
		users.POST("/:id/api-keys/rotate", middleware.RBACMiddleware("user:update"), h.rotateUserAPIKey)
		users.PUT("/:id/api-keys/:key_id/disable", middleware.RBACMiddleware("user:update"), h.disableUserAPIKey)
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

type userWriteRequest struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	DepartmentID *uint  `json:"department_id"`
	Timezone     string `json:"timezone"`
}

func (req *userWriteRequest) toUser() *admin.User {
	return &admin.User{
		Username:     req.Username,
		Email:        req.Email,
		Password:     req.Password,
		Name:         req.Name,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		Timezone:     req.Timezone,
	}
}

// createUser 创建用户
func (h *UserHandler) createUser(c *gin.Context) {
	var req userWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidUserData)
		return
	}

	user := req.toUser()
	if err := h.userService.CreateUser(user); err != nil {
		if errors.Is(err, common.ErrWeakPassword) {
			response.Error(c, response.CodePasswordError, response.MsgPasswordDoesNotMeetPolicy)
			return
		}
		response.InternalServerError(c, response.MsgFailedCreateUser)
		return
	}

	user.Password = ""
	response.Created(c, response.MsgUserCreatedSuccessfully, user)
}

// updateUser 更新用户
func (h *UserHandler) updateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}

	var req struct {
		Username     string `json:"username"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		Name         string `json:"name"`
		Phone        string `json:"phone"`
		DepartmentID *uint  `json:"department_id"`
		Timezone     string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidUserData)
		return
	}

	user := &admin.User{
		ID:           uint(id),
		Username:     req.Username,
		Email:        req.Email,
		Password:     req.Password,
		Name:         req.Name,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		Timezone:     req.Timezone,
	}
	if err := h.userService.UpdateUser(user); err != nil {
		if errors.Is(err, common.ErrWeakPassword) {
			response.Error(c, response.CodePasswordError, response.MsgPasswordDoesNotMeetPolicy)
			return
		}
		if errors.Is(err, common.ErrUserNotFound) {
			response.NotFound(c, response.MsgUserNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedUpdateUser)
		return
	}

	user.Password = ""
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
		if errors.Is(err, common.ErrUserNotFound) {
			response.NotFound(c, response.MsgUserNotFound)
			return
		}
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
		if errors.Is(err, common.ErrUserNotFound) {
			response.NotFound(c, response.MsgUserNotFound)
			return
		}
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
		if errors.Is(err, common.ErrUserNotFound) {
			response.NotFound(c, response.MsgUserNotFound)
			return
		}
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

func (h *UserHandler) getPendingRegistrationCount(c *gin.Context) {
	regSvc, ok := h.userService.(registrationService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	total, err := regSvc.CountPendingRegistrations()
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveRegistrations)
		return
	}
	response.SuccessData(c, response.MsgPendingRegistrationsRetrievedSuccessfully, gin.H{"pending_count": total})
}

func (h *UserHandler) listPendingRegistrations(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)
	regSvc, ok := h.userService.(registrationService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	list, total, err := regSvc.ListPendingRegistrations(page, pageSize)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveRegistrations)
		return
	}
	response.SuccessPaginated(c, response.MsgPendingRegistrationsRetrievedSuccessfully, list, total, page, pageSize)
}

func (h *UserHandler) reviewRegistration(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidRegistrationID)
		return
	}
	var req struct {
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if !req.Approved && strings.TrimSpace(req.Reason) == "" {
		response.BadRequest(c, response.MsgReviewReasonRequired)
		return
	}

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

	regSvc, ok := h.userService.(registrationService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	result, err := regSvc.ReviewRegistration(uint(id), uint(operatorID), req.Approved, req.Reason)
	if err != nil {
		if errors.Is(err, common.ErrRegistrationAlreadyReviewed) {
			response.Error(c, response.CodeResourceInUse, response.MsgRegistrationAlreadyReviewed)
			return
		}
		if common.IsDuplicateEntryError(err) {
			response.Error(c, response.CodeDuplicateEntry, response.MsgRegistrationConflictOnApprove)
			return
		}
		response.InternalServerError(c, response.MsgFailedReviewRegistration)
		return
	}
	h.sendReviewNotification(result, req.Reason)
	response.SuccessData(c, response.MsgRegistrationReviewedSuccessfully, result)
}

func (h *UserHandler) listRegistrationAudits(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidRegistrationID)
		return
	}
	regSvc, ok := h.userService.(registrationService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	audits, err := regSvc.ListRegistrationAudits(uint(id))
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveRegistrationAudits)
		return
	}
	response.SuccessData(c, response.MsgRegistrationAuditsRetrievedSuccessfully, audits)
}

func (h *UserHandler) sendReviewNotification(result *admin.RegistrationReviewResult, reason string) {
	if h.emailSender == nil || result == nil {
		return
	}
	var subject string
	var body string
	if result.Approved {
		subject = "注册审核通过"
		body = fmt.Sprintf("您好，%s：\n\n您的注册申请已通过审核。\n账号：%s\n初始密码：%s\nAPI Key：%s\n\n请尽快登录后修改密码并妥善保管密钥。", result.Username, result.Username, result.InitialPassword(), result.APIKey())
	} else {
		subject = "注册审核未通过"
		body = fmt.Sprintf("您好，%s：\n\n您的注册申请未通过审核。\n原因：%s\n\n如需帮助请联系管理员。", result.Username, strings.TrimSpace(reason))
	}
	if err := h.emailSender.Send(result.Email, subject, body); err != nil {
		log.Printf("warning: failed to send review notification to %s: %v", result.Email, err)
	}
}

func (h *UserHandler) listUserAPIKeys(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}
	keySvc, ok := h.userService.(apiKeyService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	keys, err := keySvc.ListUserAPIKeys(uint(userID))
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveAPIKeys)
		return
	}
	response.SuccessData(c, response.MsgAPIKeysRetrievedSuccessfully, keys)
}

func (h *UserHandler) rotateUserAPIKey(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}
	var req struct {
		Name          string `json:"name"`
		Scope         string `json:"scope"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			response.BadRequest(c, response.MsgInvalidRequestData)
			return
		}
		// 空 body 允许，走默认值
		req.Name = "default"
	}
	if req.ExpiresInDays < 0 {
		response.BadRequest(c, response.MsgInvalidExpiresInDays)
		return
	}
	if req.ExpiresInDays > 3650 {
		response.BadRequest(c, response.MsgInvalidExpiresInDaysTooLarge)
		return
	}
	keySvc, ok := h.userService.(apiKeyService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		v := time.Now().UTC().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &v
	}
	rawKey, err := keySvc.RotateUserAPIKey(uint(userID), req.Name, req.Scope, expiresAt)
	if err != nil {
		if errors.Is(err, common.ErrInvalidArgument) {
			response.BadRequest(c, response.MsgInvalidAPIKeyScope)
			return
		}
		if common.IsDuplicateEntryError(err) {
			response.Error(c, response.CodeDuplicateEntry, response.MsgFailedRotateAPIKey)
			return
		}
		response.InternalServerError(c, response.MsgFailedRotateAPIKey)
		return
	}
	response.SuccessData(c, response.MsgAPIKeyRotatedSuccessfully, gin.H{"api_key": rawKey})
}

func (h *UserHandler) disableUserAPIKey(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidUserID)
		return
	}
	keyID, err := strconv.ParseUint(c.Param("key_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidAPIKeyID)
		return
	}
	keySvc, ok := h.userService.(apiKeyService)
	if !ok {
		response.InternalServerError(c, response.MsgServiceUnavailableRetry)
		return
	}
	if err := keySvc.DisableUserAPIKey(uint(userID), uint(keyID)); err != nil {
		if errors.Is(err, common.ErrAPIKeyNotFound) {
			response.NotFound(c, response.MsgAPIKeyNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedDisableAPIKey)
		return
	}
	response.Success(c, response.MsgAPIKeyDisabledSuccessfully)
}
