package auth

import (
	"strconv"
	"time"

	"ad-platform/internal/admin"
	"ad-platform/pkg/auth"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关处理器
type AuthHandler struct {
	userService    *admin.UserService
	authMiddleware *auth.AuthMiddleware
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService *admin.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userService:    userService,
		authMiddleware: auth.NewAuthMiddleware(jwtSecret),
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
		response.Forbidden(c, response.MsgUserDisabled)
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
	token, err := h.authMiddleware.GenerateJWT(strconv.Itoa(int(user.ID)), roleName, userTimezone, 24*time.Hour)
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
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	// 创建用户
	user := &admin.User{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Status:   true,
	}

	if err := h.userService.CreateUser(user); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateUser)
		return
	}

	response.Created(c, response.MsgUserCreatedSuccessfully, user)
}
