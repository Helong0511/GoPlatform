package auth

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/notify"
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
func NewAuthHandler(userService *admin.UserService, jwtSecret, jwtIssuer, jwtAudience string, tokenTTL time.Duration, allowPublicRegister bool, emailSender ...notify.EmailSender) *AuthHandler {
	if tokenTTL <= 0 {
		tokenTTL = time.Hour
	}
	var sender notify.EmailSender
	if len(emailSender) > 0 {
		sender = emailSender[0]
	}
	return &AuthHandler{
		userService:         userService,
		authMiddleware:      auth.NewAuthMiddleware(jwtSecret, jwtIssuer, jwtAudience),
		tokenTTL:            tokenTTL,
		allowPublicRegister: allowPublicRegister,
		emailSender:         sender,
	}
}

// SetupRoutes 设置认证相关路由；loginRate/registerRate 非 nil 时分别用于登录/注册专用限流。
func (h *AuthHandler) SetupRoutes(public *gin.RouterGroup, protected *gin.RouterGroup, loginRate, registerRate gin.HandlerFunc) {
	loginChain := make([]gin.HandlerFunc, 0, 2)
	if loginRate != nil {
		loginChain = append(loginChain, loginRate)
	}
	loginChain = append(loginChain, h.login)
	public.POST("/login", loginChain...)

	regChain := make([]gin.HandlerFunc, 0, 2)
	if registerRate != nil {
		regChain = append(regChain, registerRate)
	}
	regChain = append(regChain, h.register)
	public.POST("/register", regChain...)

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

	// 生成 JWT：写入全部角色（去重保序），首个角色同时写入 role 声明以兼容旧客户端
	seen := make(map[string]struct{}, len(user.Roles))
	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		n := strings.TrimSpace(r.Name)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		roleNames = append(roleNames, n)
	}
	// 获取用户时区，默认使用 Asia/Shanghai
	userTimezone := user.Timezone
	if userTimezone == "" {
		userTimezone = "Asia/Shanghai"
	}
	token, err := h.authMiddleware.GenerateJWT(strconv.Itoa(int(user.ID)), userTimezone, roleNames, h.tokenTTL)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedGenerateToken)
		return
	}

	permSeen := make(map[string]struct{})
	permissions := make([]string, 0)
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			if _, ok := permSeen[perm.Name]; ok {
				continue
			}
			permSeen[perm.Name] = struct{}{}
			permissions = append(permissions, perm.Name)
		}
	}

	responseData := gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"name":        user.Name,
			"phone":       user.Phone,
			"timezone":    user.Timezone,
			"roles":       roleNames,
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
			response.Created(c, response.MsgRegistrationSubmitted, gin.H{
				"status": admin.RegistrationStatusPending,
			})
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
