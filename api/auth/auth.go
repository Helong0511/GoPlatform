package auth

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/notify"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关处理器
type AuthHandler struct {
	userService         *admin.UserService
	authMiddleware      *auth.AuthMiddleware
	tokenTTL            time.Duration
	allowPublicRegister bool
	emailSender         notify.EmailSender
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService *admin.UserService, jwtSecret string, tokenTTL time.Duration, allowPublicRegister bool, emailSender ...notify.EmailSender) *AuthHandler {
	if tokenTTL <= 0 {
		tokenTTL = time.Hour
	}
	var sender notify.EmailSender
	if len(emailSender) > 0 {
		sender = emailSender[0]
	}
	return &AuthHandler{
		userService:         userService,
		authMiddleware:      auth.NewAuthMiddleware(jwtSecret),
		tokenTTL:            tokenTTL,
		allowPublicRegister: allowPublicRegister,
		emailSender:         sender,
	}
}

// SetupRoutes 设置认证相关路由
func (h *AuthHandler) SetupRoutes(public *gin.RouterGroup, protected *gin.RouterGroup) {
	// 公开接口
	public.POST("/login", h.login)
	public.POST("/register", h.register)

	// 应用认证中间件
	protected.Use(h.authMiddleware.AuthMiddleware())
}

// login 登录
func (h *AuthHandler) login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	// 验证用户
	user, err := h.userService.GetUserByUsername(req.Username)
	if err != nil {
		response.Unauthorized(c, response.MsgInvalidUsernameOrPassword)
		return
	}

	// 检查密码
	if !h.userService.CheckPassword(user, req.Password) {
		response.Unauthorized(c, response.MsgInvalidUsernameOrPassword)
		return
	}

	// 检查用户状态
	if !user.Status {
		response.Forbidden(c, response.MsgUserLockedCannotLogin)
		return
	}

	// 生成JWT
	roleName := ""
	if len(user.Roles) > 0 {
		roleName = user.Roles[0].Name
	}
	// 获取用户时区，默认使用 Asia/Shanghai
	userTimezone := user.Timezone
	if userTimezone == "" {
		userTimezone = "Asia/Shanghai"
	}
	token, err := h.authMiddleware.GenerateJWT(strconv.Itoa(int(user.ID)), roleName, userTimezone, h.tokenTTL)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedGenerateToken)
		return
	}

	// 提取用户的所有权限
	var permissions []string
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permissions = append(permissions, perm.Name)
		}
	}

	// 返回统一格式的响应
	responseData := gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"name":        user.Name,
			"phone":       user.Phone,
			"timezone":    user.Timezone,
			"roles":       user.Roles,
			"permissions": permissions,
		},
	}

	response.SuccessData(c, response.MsgLoginSuccessful, responseData)
}

// register 注册
func (h *AuthHandler) register(c *gin.Context) {
	if !h.allowPublicRegister {
		response.Forbidden(c, response.MsgRegistrationDisabled)
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	reg, err := h.userService.CreateRegistration(req.Username, req.Email)
	if err != nil {
		if common.IsDuplicateEntryError(err) {
			response.Error(c, response.CodeDuplicateEntry, response.MsgRegistrationAlreadyExists)
			return
		}
		response.InternalServerError(c, response.MsgFailedCreateUser)
		return
	}

	if h.emailSender != nil {
		subject := "注册申请已提交"
		body := fmt.Sprintf("您好，%s：\n\n您的注册申请已提交，等待管理员审核。\n账号：%s\n邮箱：%s\n\n审核通过后系统会发送初始密码与API Key。", strings.TrimSpace(req.Username), strings.TrimSpace(req.Username), strings.TrimSpace(req.Email))
		if sendErr := h.emailSender.Send(req.Email, subject, body); sendErr != nil {
			log.Printf("warning: failed to send registration submitted email to %s: %v", req.Email, sendErr)
		}
	}

	response.Created(c, response.MsgRegistrationSubmitted, gin.H{
		"id":       reg.ID,
		"username": reg.Username,
		"email":    reg.Email,
		"status":   reg.Status,
	})
}
