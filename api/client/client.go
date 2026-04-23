package client

import (
	"errors"
	"strconv"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// ClientHandler 客户管理处理器
type ClientHandler struct {
	userService *admin.UserService
}

// NewClientHandler 创建客户处理器
func NewClientHandler(userService *admin.UserService) *ClientHandler {
	return &ClientHandler{
		userService: userService,
	}
}

// SetupRoutes 设置客户相关路由
func (h *ClientHandler) SetupRoutes(protected *gin.RouterGroup) {
	// 客户管理
	clients := protected.Group("/clients")
	{
		clients.GET("", middleware.RBACMiddleware("client:read"), h.listClients)
		clients.GET("/:id", middleware.RBACMiddleware("client:read"), h.getClient)
		clients.POST("", middleware.RBACMiddleware("client:create"), h.createClient)
		clients.PUT("/:id", middleware.RBACMiddleware("client:update"), h.updateClient)
		clients.DELETE("/:id", middleware.RBACMiddleware("client:delete"), h.deleteClient)
	}
}

// listClients 列出客户
func (h *ClientHandler) listClients(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	clients, total, err := h.userService.ListClients(page, pageSize)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveClients)
		return
	}

	response.SuccessPaginated(c, response.MsgClientsRetrievedSuccessfully, clients, total, page, pageSize)
}

// getClient 获取客户
func (h *ClientHandler) getClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidClientID)
		return
	}

	client, err := h.userService.GetClientByID(uint(id))
	if err != nil {
		if errors.Is(err, common.ErrClientNotFound) {
			response.NotFound(c, response.MsgClientNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedRetrieveClient)
		return
	}

	response.SuccessData(c, response.MsgClientRetrievedSuccessfully, client)
}

// createClient 创建客户
func (h *ClientHandler) createClient(c *gin.Context) {
	var client admin.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		response.BadRequest(c, response.MsgInvalidClientData)
		return
	}

	if err := h.userService.CreateClient(&client); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateClient)
		return
	}

	response.Created(c, response.MsgClientCreatedSuccessfully, client)
}

// updateClient 更新客户
func (h *ClientHandler) updateClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidClientID)
		return
	}

	var client admin.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		response.BadRequest(c, response.MsgInvalidClientData)
		return
	}

	client.ID = uint(id)
	if err := h.userService.UpdateClient(&client); err != nil {
		response.InternalServerError(c, response.MsgFailedUpdateClient)
		return
	}

	response.SuccessData(c, response.MsgClientUpdatedSuccessfully, client)
}

// deleteClient 删除客户
func (h *ClientHandler) deleteClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidClientID)
		return
	}

	if err := h.userService.DeleteClient(uint(id)); err != nil {
		response.InternalServerError(c, response.MsgFailedDeleteClient)
		return
	}

	response.Success(c, response.MsgClientDeletedSuccessfully)
}
