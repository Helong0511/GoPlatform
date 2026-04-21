package department

import (
	"errors"
	"strconv"

	"ad-platform/internal/common"
	"ad-platform/internal/admin"
	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

// DepartmentHandler 部门管理处理器
type DepartmentHandler struct {
	departmentService *admin.DepartmentService
}

// NewDepartmentHandler 创建部门处理器
func NewDepartmentHandler(departmentService *admin.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{
		departmentService: departmentService,
	}
}

// SetupRoutes 设置部门相关路由
func (h *DepartmentHandler) SetupRoutes(protected *gin.RouterGroup) {
	// 部门管理
	departments := protected.Group("/departments")
	{
		departments.GET("", h.listDepartments)
		departments.GET("/:id", h.getDepartment)
		departments.POST("", h.createDepartment)
		departments.PUT("/:id", h.updateDepartment)
		departments.DELETE("/:id", h.deleteDepartment)
	}
}

// listDepartments 列出部门
func (h *DepartmentHandler) listDepartments(c *gin.Context) {
	onlyActive := c.Query("active") == "true"
	departments, err := h.departmentService.ListDepartments(onlyActive)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveDepartments)
		return
	}

	response.SuccessData(c, response.MsgDepartmentsRetrievedSuccessfully, departments)
}

// getDepartment 获取部门
func (h *DepartmentHandler) getDepartment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidDepartmentID)
		return
	}

	dept, err := h.departmentService.GetDepartmentByID(uint(id))
	if err != nil {
		if errors.Is(err, common.ErrDepartmentNotFound) {
			response.NotFound(c, response.MsgDepartmentNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedRetrieveDepartment)
		return
	}

	response.SuccessData(c, response.MsgDepartmentRetrievedSuccessfully, dept)
}

// createDepartment 创建部门
func (h *DepartmentHandler) createDepartment(c *gin.Context) {
	var dept admin.Department
	if err := c.ShouldBindJSON(&dept); err != nil {
		response.BadRequest(c, response.MsgInvalidDepartmentData)
		return
	}

	if err := h.departmentService.CreateDepartment(&dept); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateDepartment)
		return
	}

	response.Created(c, response.MsgDepartmentCreatedSuccessfully, dept)
}

// updateDepartment 更新部门
func (h *DepartmentHandler) updateDepartment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidDepartmentID)
		return
	}

	var dept admin.Department
	if err := c.ShouldBindJSON(&dept); err != nil {
		response.BadRequest(c, response.MsgInvalidDepartmentData)
		return
	}

	dept.ID = uint(id)
	if err := h.departmentService.UpdateDepartment(&dept); err != nil {
		response.InternalServerError(c, response.MsgFailedUpdateDepartment)
		return
	}

	response.SuccessData(c, response.MsgDepartmentUpdatedSuccessfully, dept)
}

// deleteDepartment 删除部门
func (h *DepartmentHandler) deleteDepartment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidDepartmentID)
		return
	}

	if err := h.departmentService.ArchiveDepartment(uint(id)); err != nil {
		response.InternalServerError(c, response.MsgFailedArchiveDepartment)
		return
	}

	response.Success(c, response.MsgDepartmentArchivedSuccessfully)
}
