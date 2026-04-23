package ad

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	ad "ad-platform/internal/ad"
	"ad-platform/internal/common"
	"ad-platform/internal/security"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// AdHandler 广告管理处理器
type AdHandler struct {
	adService *ad.AdService
	aesUtils  *security.AESUtils
}

// NewAdHandler 创建广告处理器
func NewAdHandler(adService *ad.AdService, aesUtils *security.AESUtils) *AdHandler {
	return &AdHandler{
		adService: adService,
		aesUtils:  aesUtils,
	}
}

// SetupRoutes 设置广告相关路由
func (h *AdHandler) SetupRoutes(engine *gin.Engine, public *gin.RouterGroup, protected *gin.RouterGroup) {
	// 公开接口 - 只支持POST
	public.POST("/getWebTask", h.getAds)

	// 广告管理
	ads := protected.Group("/ads")
	{
		ads.GET("", middleware.RBACMiddleware("ad:read"), h.listAdTasks)
		ads.GET("/active", middleware.RBACMiddleware("ad:read"), h.listActiveAdTasks)
		ads.GET("/:id", middleware.RBACMiddleware("ad:read"), h.getAdTask)
		ads.POST("", middleware.RBACMiddleware("ad:create"), h.createAdTask)
		ads.PUT("/:id", middleware.RBACMiddleware("ad:update"), h.updateAdTask)
		ads.DELETE("/:id", middleware.RBACMiddleware("ad:delete"), h.deleteAdTask)
	}
}

// listAdTasks 列出广告任务
// 后端只返回UTC标准时间，前端负责时区转换
func (h *AdHandler) listAdTasks(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	ads, total, err := h.adService.ListAdTasks(page, pageSize)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveAdTasks)
		return
	}

	// 后端只返回UTC标准时间，不进行时区转换
	response.SuccessPaginated(c, response.MsgAdTasksRetrievedSuccessfully, ads, total, page, pageSize)
}

// getAdTask 获取广告任务
// 后端只返回UTC标准时间，前端负责时区转换
func (h *AdHandler) getAdTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidAdID)
		return
	}

	adTask, err := h.adService.GetAdTaskByID(uint(id))
	if err != nil {
		if errors.Is(err, common.ErrAdTaskNotFound) {
			response.NotFound(c, response.MsgAdTaskNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedRetrieveAdTask)
		return
	}

	// 后端只返回UTC标准时间，不进行时区转换
	response.SuccessData(c, response.MsgAdTaskRetrievedSuccessfully, adTask)
}

// createAdTask 创建广告任务
// 前端传入UTC标准时间，后端直接保存
func (h *AdHandler) createAdTask(c *gin.Context) {
	var adTask ad.AdTask
	if err := c.ShouldBindJSON(&adTask); err != nil {
		response.BadRequest(c, response.MsgInvalidAdTaskData)
		return
	}

	// 验证必填字段
	if adTask.Name == "" {
		response.BadRequest(c, response.MsgAdNameRequired)
		return
	}
	if adTask.StartTime.IsZero() {
		response.BadRequest(c, response.MsgStartTimeRequired)
		return
	}
	if adTask.EndTime.IsZero() {
		response.BadRequest(c, response.MsgEndTimeRequired)
		return
	}
	if adTask.EndTime.Before(adTask.StartTime) {
		response.BadRequest(c, response.MsgEndTimeAfterStartRequired)
		return
	}

	// 前端传入的是 UTC 标准时间，直接保存
	if !adTask.StartTime.IsZero() {
		adTask.StartTime = adTask.StartTime.UTC()
	}
	if !adTask.EndTime.IsZero() {
		adTask.EndTime = adTask.EndTime.UTC()
	}

	if err := h.adService.CreateAdTask(&adTask); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateAdTask)
		return
	}

	// 后端只返回UTC标准时间，不进行时区转换
	response.Created(c, response.MsgAdTaskCreatedSuccessfully, adTask)
}

// updateAdTask 更新广告任务
// 前端传入UTC标准时间，后端直接保存
func (h *AdHandler) updateAdTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidAdID)
		return
	}

	var adTask ad.AdTask
	if err := c.ShouldBindJSON(&adTask); err != nil {
		response.BadRequest(c, response.MsgInvalidAdTaskData)
		return
	}

	// 验证必填字段
	if adTask.Name == "" {
		response.BadRequest(c, response.MsgAdNameRequired)
		return
	}
	if adTask.StartTime.IsZero() {
		response.BadRequest(c, response.MsgStartTimeRequired)
		return
	}
	if adTask.EndTime.IsZero() {
		response.BadRequest(c, response.MsgEndTimeRequired)
		return
	}
	if adTask.EndTime.Before(adTask.StartTime) {
		response.BadRequest(c, response.MsgEndTimeAfterStartRequired)
		return
	}

	// 前端传入的是 UTC 标准时间，直接保存
	if !adTask.StartTime.IsZero() {
		adTask.StartTime = adTask.StartTime.UTC()
	}
	if !adTask.EndTime.IsZero() {
		adTask.EndTime = adTask.EndTime.UTC()
	}

	adTask.ID = uint(id)
	if err := h.adService.UpdateAdTask(&adTask); err != nil {
		response.InternalServerError(c, response.MsgFailedUpdateAdTask)
		return
	}

	// 后端只返回UTC标准时间，不进行时区转换
	response.SuccessData(c, response.MsgAdTaskUpdatedSuccessfully, adTask)
}

// deleteAdTask 删除广告任务
func (h *AdHandler) deleteAdTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidAdID)
		return
	}

	if err := h.adService.DeleteAdTask(uint(id)); err != nil {
		response.InternalServerError(c, response.MsgFailedDeleteAdTask)
		return
	}

	response.Success(c, response.MsgAdTaskDeletedSuccessfully)
}

// listActiveAdTasks 列出活跃的广告任务
// 后端只返回UTC标准时间，前端负责时区转换
func (h *AdHandler) listActiveAdTasks(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	// 构建筛选参数
	filters := make(map[string]interface{})
	validationDetails := make([]response.ErrorDetail, 0)
	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if startTime, ok, detail := util.ParseOptionalRFC3339Query("start_time", c.Query("start_time")); detail != nil {
		validationDetails = append(validationDetails, *detail)
	} else if ok {
		filters["start_time"] = startTime
	}
	if endTime, ok, detail := util.ParseOptionalRFC3339Query("end_time", c.Query("end_time")); detail != nil {
		validationDetails = append(validationDetails, *detail)
	} else if ok {
		filters["end_time"] = endTime
	}

	startTime, hasStartTime := filters["start_time"].(time.Time)
	endTime, hasEndTime := filters["end_time"].(time.Time)
	if hasStartTime && hasEndTime {
		if detail := util.ValidateTimeRange(startTime, endTime); detail != nil {
			validationDetails = append(validationDetails, *detail)
		}
	}

	// 构建排序参数
	sortField := c.DefaultQuery("sort_field", "id")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	sortValidation := util.ValidateSortParams(sortField, sortOrder, map[string]struct{}{
		"id":         {},
		"name":       {},
		"priority":   {},
		"created_at": {},
		"start_time": {},
		"end_time":   {},
	})
	validationDetails = append(validationDetails, sortValidation...)

	if len(validationDetails) > 0 {
		response.ErrorValidation(c, response.MsgInvalidQueryParameters, validationDetails)
		return
	}

	ads, total, err := h.adService.ListActiveAdTasks(page, pageSize, filters, sortField, sortOrder)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveAdTasks)
		return
	}

	// 后端只返回UTC标准时间，不进行时区转换
	response.SuccessPaginated(c, response.MsgActiveAdTasksRetrievedSuccessfully, ads, total, page, pageSize)
}

// getAds 获取广告任务（外部接口，只支持POST）
// 后端只返回UTC标准时间，前端负责时区转换
func (h *AdHandler) getAds(c *gin.Context) {
	// 获取活跃的广告任务
	ads, err := h.adService.GetActiveAdTasks()
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveAdTasks)
		return
	}

	// 将任务数据序列化为JSON
	adsJSON, err := json.Marshal(ads)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedSerializeData)
		return
	}

	// 加密数据
	encryptedData, err := h.aesUtils.Encrypt(string(adsJSON))
	if err != nil {
		response.InternalServerError(c, response.MsgFailedEncryptData)
		return
	}

	// 返回加密后的响应
	responseData := gin.H{
		"msg":  response.MsgSuccessLowercase,
		"code": 200,
		"data": encryptedData,
		"jsc":  "", // 广告位判断js代码
		"jscd": "", // 跳转详情页js代码
		"jsch": "", // 搜索新增, 获取content height的js代码
	}

	response.SuccessData(c, response.MsgSuccess, responseData)
}
