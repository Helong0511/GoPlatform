package report

import (
	"time"

	report "ad-platform/internal/report"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

const maxBatchReports = 100

// ReportHandler 报告管理处理器
type ReportHandler struct {
	reportService reportService
}

type reportService interface {
	ReportAd(report *report.AdReport) error
	GetTodayReportsByTimezone(timezone string) (int64, error)
	GetReportStatsByTimezone(timezone string) (map[string]interface{}, error)
	ListAdReportsWithFilter(page, pageSize int, filters map[string]interface{}) ([]report.AdReport, int64, error)
}

// NewReportHandler 创建报告处理器
func NewReportHandler(reportService *report.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// NewReportHandlerWithService 用于测试和自定义实现注入。
func NewReportHandlerWithService(reportService reportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// SetupRoutes 设置报告相关路由
func (h *ReportHandler) SetupRoutes(public *gin.RouterGroup, protected *gin.RouterGroup, apiKeyGuard ...gin.HandlerFunc) {
	var reportWriteGuard gin.HandlerFunc
	if len(apiKeyGuard) > 0 && apiKeyGuard[0] != nil {
		reportWriteGuard = apiKeyGuard[0]
	}
	if reportWriteGuard != nil {
		public.POST("/report", reportWriteGuard, h.reportAd)
	} else {
		public.POST("/report", h.reportAd)
	}

	reports := protected.Group("/reports")
	{
		reports.GET("/today", middleware.RBACMiddleware("report:read"), h.getTodayReports)
		reports.GET("/stats", middleware.RBACMiddleware("report:read"), h.getReportStats)
		reports.GET("", middleware.RBACMiddleware("report:read"), h.listAdReports)
		reports.POST("/batch", middleware.RBACMiddleware("report:create"), h.batchCreateAdReports)
	}
}

// reportAd 上报广告执行数据（默认由路由挂载 API Key，需 report:write）
func (h *ReportHandler) reportAd(c *gin.Context) {
	var body report.AdReport
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, response.MsgInvalidReportData)
		return
	}
	if body.Action == "" {
		response.BadRequest(c, response.MsgInvalidReportData)
		return
	}

	body.Timestamp = time.Now().UTC()

	if err := h.reportService.ReportAd(&body); err != nil {
		response.InternalServerError(c, response.MsgFailedSubmitReport)
		return
	}

	response.Success(c, response.MsgReportSubmittedSuccessfully)
}

func (h *ReportHandler) getTodayReports(c *gin.Context) {
	timezone := c.GetString("timezone")
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	count, err := h.reportService.GetTodayReportsByTimezone(timezone)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveTodayReports)
		return
	}

	response.SuccessData(c, response.MsgTodayReportsRetrievedSuccessfully, gin.H{"count": count})
}

func (h *ReportHandler) getReportStats(c *gin.Context) {
	timezone := c.GetString("timezone")
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	stats, err := h.reportService.GetReportStatsByTimezone(timezone)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveReportStats)
		return
	}

	response.SuccessData(c, response.MsgReportStatsRetrievedSuccessfully, stats)
}

func (h *ReportHandler) listAdReports(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	filters := make(map[string]interface{})
	validationDetails := make([]response.ErrorDetail, 0)

	if adID, ok, detail := util.ParseOptionalUintQuery("ad_id", c.Query("ad_id")); detail != nil {
		validationDetails = append(validationDetails, *detail)
	} else if ok {
		filters["ad_id"] = adID
	}

	if action := c.Query("action"); action != "" {
		filters["action"] = action
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

	if len(validationDetails) > 0 {
		response.ErrorValidation(c, response.MsgInvalidQueryParameters, validationDetails)
		return
	}

	reports, total, err := h.reportService.ListAdReportsWithFilter(page, pageSize, filters)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveAdReports)
		return
	}

	response.SuccessPaginated(c, response.MsgAdReportsRetrievedSuccessfully, reports, total, page, pageSize)
}

func (h *ReportHandler) batchCreateAdReports(c *gin.Context) {
	var req struct {
		Reports []report.AdReport `json:"reports" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if len(req.Reports) == 0 {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if len(req.Reports) > maxBatchReports {
		response.BadRequest(c, response.MsgBatchReportsTooLarge)
		return
	}

	for i := range req.Reports {
		if req.Reports[i].Action == "" {
			response.BadRequest(c, response.MsgInvalidReportData)
			return
		}
		req.Reports[i].Timestamp = time.Now().UTC()
		if err := h.reportService.ReportAd(&req.Reports[i]); err != nil {
			response.InternalServerError(c, response.MsgFailedBatchSubmitReports)
			return
		}
	}

	response.SuccessData(c, response.MsgBatchReportsSubmittedSuccessfully, gin.H{"count": len(req.Reports)})
}
