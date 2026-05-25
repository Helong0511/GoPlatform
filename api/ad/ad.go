package ad

import (
	"errors"
	"strconv"
	"strings"
	"time"

	ad "ad-platform/internal/ad"
	"ad-platform/internal/common"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// AdHandler 广告管理处理器
type AdHandler struct {
	adService *ad.AdService
}

// NewAdHandler 创建广告处理器
func NewAdHandler(adService *ad.AdService) *AdHandler {
	return &AdHandler{
		adService: adService,
	}
}

// SetupRoutes 设置广告管理路由（仅管理端）
func (h *AdHandler) SetupRoutes(protected *gin.RouterGroup) {
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

type adTaskWriteRequest struct {
	Name               string    `json:"name" binding:"required"`
	Description        string    `json:"description"`
	ClientID           *uint     `json:"client_id"`
	Status             string    `json:"status"`
	Priority           int       `json:"priority"`
	StartTime          time.Time `json:"start_time" binding:"required"`
	EndTime            time.Time `json:"end_time" binding:"required"`
	TargetAudience     string    `json:"target_audience"`
	Creative           string    `json:"creative"`
	Budget             float64   `json:"budget"`
	ProxyStrategy      int       `json:"proxy_strategy"`
	BizType            int       `json:"biz_type"`
	Profile            string    `json:"profile"`
	Platform           string    `json:"plat"`
	CAU                int       `json:"cau"`
	OS                 int       `json:"os"`
	InterceptInfo      string    `json:"intercept_info"`
	LandURL            string    `json:"landurl"`
	AllowCounty        string    `json:"allow_county"`
	FinalDomains       string    `json:"final_domains"`
	Info               string    `json:"info"`
	UserQuality        string    `json:"user_quality"`
	ClickHomeOffsetX   int       `json:"click_home_offset_x"`
	ClickHomeOffsetY   int       `json:"click_home_offset_y"`
	ClickResultOffsetX int       `json:"click_result_offset_x"`
	ClickResultOffsetY int       `json:"click_result_offset_y"`
	FKeyAPI            string    `json:"f_key_api"`
	PlanID             *int      `json:"plan_id"`
}

func (req *adTaskWriteRequest) toAdTask() *ad.AdTask {
	return &ad.AdTask{
		Name:               req.Name,
		Description:        req.Description,
		ClientID:           req.ClientID,
		Status:             req.Status,
		Priority:           req.Priority,
		StartTime:          req.StartTime.UTC(),
		EndTime:            req.EndTime.UTC(),
		TargetAudience:     req.TargetAudience,
		Creative:           req.Creative,
		Budget:             req.Budget,
		ProxyStrategy:      req.ProxyStrategy,
		BizType:            req.BizType,
		Profile:            req.Profile,
		Platform:           req.Platform,
		CAU:                req.CAU,
		OS:                 req.OS,
		InterceptInfo:      req.InterceptInfo,
		LandURL:            req.LandURL,
		AllowCounty:        req.AllowCounty,
		FinalDomains:       req.FinalDomains,
		Info:               req.Info,
		UserQuality:        req.UserQuality,
		ClickHomeOffsetX:   req.ClickHomeOffsetX,
		ClickHomeOffsetY:   req.ClickHomeOffsetY,
		ClickResultOffsetX: req.ClickResultOffsetX,
		ClickResultOffsetY: req.ClickResultOffsetY,
		FKeyAPI:            req.FKeyAPI,
		PlanID:             req.PlanID,
	}
}

func validateAdTaskTimes(start, end time.Time) bool {
	return !start.IsZero() && !end.IsZero() && !end.Before(start)
}

// listAdTasks 列出广告任务
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

	response.SuccessPaginated(c, response.MsgAdTasksRetrievedSuccessfully, ads, total, page, pageSize)
}

// getAdTask 获取广告任务
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

	response.SuccessData(c, response.MsgAdTaskRetrievedSuccessfully, adTask)
}

// createAdTask 创建广告任务
func (h *AdHandler) createAdTask(c *gin.Context) {
	var req adTaskWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidAdTaskData)
		return
	}
	if !validateAdTaskTimes(req.StartTime, req.EndTime) {
		response.BadRequest(c, response.MsgEndTimeAfterStartRequired)
		return
	}

	adTask := req.toAdTask()
	if err := h.adService.CreateAdTask(adTask); err != nil {
		response.InternalServerError(c, response.MsgFailedCreateAdTask)
		return
	}

	response.Created(c, response.MsgAdTaskCreatedSuccessfully, adTask)
}

// updateAdTask 更新广告任务
func (h *AdHandler) updateAdTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, response.MsgInvalidAdID)
		return
	}

	var req adTaskWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidAdTaskData)
		return
	}
	if !validateAdTaskTimes(req.StartTime, req.EndTime) {
		response.BadRequest(c, response.MsgEndTimeAfterStartRequired)
		return
	}

	adTask := req.toAdTask()
	adTask.ID = uint(id)
	if err := h.adService.UpdateAdTask(adTask); err != nil {
		if errors.Is(err, common.ErrAdTaskNotFound) {
			response.NotFound(c, response.MsgAdTaskNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedUpdateAdTask)
		return
	}

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
		if errors.Is(err, common.ErrAdTaskNotFound) {
			response.NotFound(c, response.MsgAdTaskNotFound)
			return
		}
		response.InternalServerError(c, response.MsgFailedDeleteAdTask)
		return
	}

	response.Success(c, response.MsgAdTaskDeletedSuccessfully)
}

// listActiveAdTasks 列出活跃的广告任务
func (h *AdHandler) listActiveAdTasks(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	filters := make(map[string]interface{})
	validationDetails := make([]response.ErrorDetail, 0)
	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if strings.ToLower(status) != "active" {
			validationDetails = append(validationDetails, response.ErrorDetail{
				Field:   "status",
				Message: "status must be active or omitted (this endpoint only lists active tasks)",
			})
		}
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

	response.SuccessPaginated(c, response.MsgActiveAdTasksRetrievedSuccessfully, ads, total, page, pageSize)
}
