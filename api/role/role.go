package role

import (
	"errors"
	"strconv"

	"ad-platform/internal/common"
	"ad-platform/internal/admin"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// RoleHandler 角色管理处理器
type RoleHandler struct {
	roleService *admin.RoleService
}

// NewRoleHandler 创建角色处理器
func NewRoleHandler(roleService *admin.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

// SetupRoutes 设置角色相关路由
func (h *RoleHandler) SetupRoutes(protected *gin.RouterGroup) {
	// 角色管理
	roles := protected.Group("/roles")
	{
		roles.GET("", h.listRoles)
		roles.GET("/:id", h.getRole)
		roles.POST("", h.createRole)
		roles.PUT("/:id", h.updateRole)
		roles.DELETE("/:id", h.deleteRole)
		roles.POST("/:id/permissions", h.assignPermissions)
		roles.POST("/assign", h.assignRoles)
	}

	// 权限管理
	protected.GET("/permissions", h.listPermissions)
}

// listRoles 列出角色
func (h *RoleHandler) listRoles(c *gin.Context) {
	roles, err := h.roleService.ListRoles()
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveRoles)
		return
	}

	response.SuccessData(c, response.MsgRolesRetrievedSuccessfully, roles)
}

// getRole 获取角色
func (h *RoleHandler) getRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidRoleID)
		return
	}

	role, err := h.roleService.GetRoleByID(uint(id))
	if err != nil {
		if errors.Is(err, common.ErrRoleNotFound) {
			response.NotFound(c, response.MsgRoleNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedRetrieveRole)
		return
	}

	response.SuccessData(c, response.MsgRoleRetrievedSuccessfully, role)
}

// createRole 创建角色
func (h *RoleHandler) createRole(c *gin.Context) {
	var role admin.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		response.BadRequest(c, response.MsgInvalidRoleData)
		return
	}

	if err := h.roleService.CreateRole(&role); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateRole)
		return
	}

	response.Created(c, response.MsgRoleCreatedSuccessfully, role)
}

// updateRole 更新角色
func (h *RoleHandler) updateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidRoleID)
		return
	}

	var role admin.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		response.BadRequest(c, response.MsgInvalidRoleData)
		return
	}

	role.ID = uint(id)
	if err := h.roleService.UpdateRole(&role); err != nil {
		response.InternalServerError(c, response.MsgFailedUpdateRole)
		return
	}

	response.SuccessData(c, response.MsgRoleUpdatedSuccessfully, role)
}

// deleteRole 删除角色
func (h *RoleHandler) deleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidRoleID)
		return
	}

	if err := h.roleService.DeleteRole(uint(id)); err != nil {
		response.InternalServerError(c, response.MsgFailedDeleteRole)
		return
	}

	response.Success(c, response.MsgRoleDeletedSuccessfully)
}

// assignPermissions 分配权限
func (h *RoleHandler) assignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidRoleID)
		return
	}

	var req struct {
		PermissionIDs []uint `json:"permission_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	if err := h.roleService.AssignPermissions(uint(id), req.PermissionIDs); err != nil {
		response.InternalServerError(c, response.MsgFailedAssignPermissions)
		return
	}

	response.Success(c, response.MsgPermissionsAssignedSuccessfully)
}

// assignRoles 分配角色
func (h *RoleHandler) assignRoles(c *gin.Context) {
	var req struct {
		UserID  uint   `json:"user_id" binding:"required"`
		RoleIDs []uint `json:"role_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	if err := h.roleService.AssignRoles(req.UserID, req.RoleIDs); err != nil {
		response.InternalServerError(c, response.MsgFailedAssignRoles)
		return
	}

	response.Success(c, response.MsgRolesAssignedSuccessfully)
}

// listPermissions 列出权限
func (h *RoleHandler) listPermissions(c *gin.Context) {
	permissions, err := h.roleService.ListPermissions()
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrievePermissions)
		return
	}

	response.SuccessData(c, response.MsgPermissionsRetrievedSuccessfully, permissions)
}
