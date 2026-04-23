package report

import (
	"encoding/json"
	"strconv"
	"time"

	report "ad-platform/internal/report"
	"ad-platform/internal/security"
	"ad-platform/pkg/middleware"
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

// ReportHandler 报告管理处理器
type ReportHandler struct {
	reportService reportService
	aesUtils      *security.AESUtils
}

type reportService interface {
	ReportAd(report *report.AdReport) error
	GetTodayReportsByTimezone(timezone string) (int64, error)
	GetReportStatsByTimezone(timezone string) (map[string]interface{}, error)
	ListAdReportsWithFilter(page, pageSize int, filters map[string]interface{}) ([]report.AdReport, int64, error)
}

// NewReportHandler 创建报告处理器
func NewReportHandler(reportService *report.ReportService, aesUtils *security.AESUtils) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		aesUtils:      aesUtils,
	}
}

// NewReportHandlerWithService 用于测试和自定义实现注入。
func NewReportHandlerWithService(reportService reportService, aesUtils *security.AESUtils) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		aesUtils:      aesUtils,
	}
}

// SetupRoutes 设置报告相关路由
func (h *ReportHandler) SetupRoutes(engine *gin.Engine, public *gin.RouterGroup, protected *gin.RouterGroup) {
	// 公开接口
	public.POST("/report", h.reportAd)

	// 外部接口 - 只支持POST
	engine.POST("/upWebTask", h.upWebTask)

	// 报告管理
	reports := protected.Group("/reports")
	{
		reports.GET("/today", middleware.RBACMiddleware("report:read"), h.getTodayReports)
		reports.GET("/stats", middleware.RBACMiddleware("report:read"), h.getReportStats)
		reports.GET("", middleware.RBACMiddleware("report:read"), h.listAdReports)
		reports.POST("/batch", middleware.RBACMiddleware("report:create"), h.batchCreateAdReports)
	}
}

// reportAd 上报广告执行数据（公开接口）
// 后端使用UTC标准时间保存
func (h *ReportHandler) reportAd(c *gin.Context) {
	var report report.AdReport
	if err := c.ShouldBindJSON(&report); err != nil {
		response.BadRequest(c, response.MsgInvalidReportData)
		return
	}

	// 设置时间戳，使用 UTC 标准时间
	report.Timestamp = time.Now().UTC()

	if err := h.reportService.ReportAd(&report); err != nil {
		response.InternalServerError(c, response.MsgFailedSubmitReport)
		return
	}

	response.Success(c, response.MsgReportSubmittedSuccessfully)
}

// getTodayReports 获取今日上报数据
// 前端传入用户时区，后端根据时区计算今日范围
func (h *ReportHandler) getTodayReports(c *gin.Context) {
	// 从上下文中获取用户时区，默认使用 Asia/Shanghai
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

// getReportStats 获取上报统计数据
// 前端传入用户时区，后端根据时区计算统计范围
func (h *ReportHandler) getReportStats(c *gin.Context) {
	// 从上下文中获取用户时区，默认使用 Asia/Shanghai
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

// listAdReports 获取广告上报列表
// 前端传入UTC标准时间，后端只处理UTC时间
// 后端只返回UTC标准时间，前端负责时区转换
func (h *ReportHandler) listAdReports(c *gin.Context) {
	page, pageSize := util.ParsePaginationQuery(
		c.DefaultQuery("page", "1"),
		c.DefaultQuery("page_size", "10"),
	)

	// 构建筛选条件
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

	// 获取上报列表
	reports, total, err := h.reportService.ListAdReportsWithFilter(page, pageSize, filters)
	if err != nil {
		response.InternalServerError(c, response.MsgFailedRetrieveAdReports)
		return
	}

	// 后端只返回UTC标准时间，不进行时区转换
	response.SuccessPaginated(c, response.MsgAdReportsRetrievedSuccessfully, reports, total, page, pageSize)
}

// batchCreateAdReports 批量添加上报数据
// 后端使用UTC标准时间保存
func (h *ReportHandler) batchCreateAdReports(c *gin.Context) {
	var req struct {
		Reports []report.AdReport `json:"reports" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	// 处理批量上报
	for i := range req.Reports {
		// 设置时间戳，使用 UTC 标准时间
		req.Reports[i].Timestamp = time.Now().UTC()

		// 上报数据
		if err := h.reportService.ReportAd(&req.Reports[i]); err != nil {
			response.InternalServerError(c, response.MsgFailedBatchSubmitReports)
			return
		}
	}

	response.SuccessData(c, response.MsgBatchReportsSubmittedSuccessfully, gin.H{"count": len(req.Reports)})
}

// upWebTask 上报广告任务执行数据
// 后端使用UTC标准时间保存
func (h *ReportHandler) upWebTask(c *gin.Context) {
	// 先解析加密数据结构
	var encryptedReq struct {
		Data string `json:"data"`
	}

	if err := c.ShouldBindJSON(&encryptedReq); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}

	// 解密数据
	if h.aesUtils == nil {
		response.InternalServerError(c, response.MsgFailedDecryptData)
		return
	}
	decryptedData, err := h.aesUtils.Decrypt(encryptedReq.Data)
	if err != nil {
		response.BadRequest(c, response.MsgFailedDecryptData)
		return
	}

	// 解析解密后的数据
	var req struct {
		ClientInner   string  `json:"client_inner"`
		ClientOuter   string  `json:"client_outer"`
		RealIP        string  `json:"real_ip"`
		IP            string  `json:"ip"`
		Click         int     `json:"click"`
		Ads           int     `json:"ads"`
		HomePage      int     `json:"homePage"`
		SubpageAccess int     `json:"subpage_access"`
		Referer       string  `json:"referer"`
		Country       string  `json:"country"`
		TID           string  `json:"TID"`
		Seq           string  `json:"Seq"`
		Error         string  `json:"error"`
		ShouldClick   bool    `json:"should_click"`
		ShouldSubpage bool    `json:"should_subpage"`
		IOS           bool    `json:"ios"`
		ClickAble     bool    `json:"click_able"`
		SourceType    string  `json:"source_type"`
		IPSource      int     `json:"ip_source"`
		AdType        int     `json:"ad_type"`
		IsReview      int     `json:"is_review"`
		LastSeq       string  `json:"last_seq"`
		ScreenWidth   int     `json:"screen_width"`
		RealURL       string  `json:"real_url"`
		OriginURL     string  `json:"origin_url"`
		IsSave        int     `json:"is_save"`
		GPUError      int     `json:"gpu_error"`
		ReqAdsCount   int     `json:"req_ads_count"`
		MatchCount    int     `json:"match_count"`
		ActiveView    int     `json:"active_view"`
		SendBytes     string  `json:"send_bytes"`
		RecvBytes     string  `json:"recv_bytes"`
		SleepInfo     string  `json:"sleep_info"`
		StartTime     int64   `json:"start_time"`
		EndTime       int64   `json:"end_time"`
		PlanID        int     `json:"PlanID"`
		Time          string  `json:"time"`
		UserID        string  `json:"userId"`
		DeviceID      string  `json:"deviceId"`
		AppID         string  `json:"appId"`
		Action        string  `json:"action"`
		AdID          string  `json:"adId"`
		DeviceInfo    string  `json:"device_info"`
		UseTime       float64 `json:"use_time"`
	}

	if err := json.Unmarshal([]byte(decryptedData), &req); err != nil {
		response.BadRequest(c, response.MsgFailedParseDecryptedData)
		return
	}

	// 构建上报数据
	report := &report.AdReport{
		UserID:        req.UserID,
		Action:        req.Action,
		DeviceInfo:    req.DeviceInfo,
		IP:            req.IP,
		ClientInner:   req.ClientInner,
		ClientOuter:   req.ClientOuter,
		RealIP:        req.RealIP,
		Click:         req.Click,
		UseTime:       req.UseTime,
		Ads:           req.Ads,
		HomePage:      req.HomePage == 1,
		SubpageAccess: req.SubpageAccess,
		Referer:       req.Referer,
		Country:       req.Country,
		Seq:           req.Seq,
		Error:         req.Error,
		ShouldClick:   req.ShouldClick,
		ShouldSubpage: req.ShouldSubpage,
		IOS:           req.IOS,
		ClickAble:     req.ClickAble,
		SourceType:    req.SourceType,
		IPSource:      req.IPSource,
		AdType:        req.AdType,
		IsReview:      req.IsReview,
		LastSeq:       req.LastSeq,
		ScreenWidth:   req.ScreenWidth,
		RealURL:       req.RealURL,
		OriginURL:     req.OriginURL,
		IsSave:        req.IsSave,
		GPUError:      req.GPUError,
		ReqAdsCount:   req.ReqAdsCount,
		MatchCount:    req.MatchCount,
		ActiveView:    req.ActiveView,
		SendBytes:     req.SendBytes,
		RecvBytes:     req.RecvBytes,
		SleepInfo:     req.SleepInfo,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		PlanID:        req.PlanID,
		Time:          req.Time,
		Timestamp:     time.Now().UTC(), // 使用 UTC 标准时间
	}

	// 尝试解析 adId 为 uint
	if req.AdID != "" {
		if adID, err := strconv.ParseUint(req.AdID, 10, 32); err == nil {
			report.AdID = uint(adID)
		}
	}

	// 上报数据
	if err := h.reportService.ReportAd(report); err != nil {
		response.InternalServerError(c, response.MsgFailedSubmitTaskReport)
		return
	}

	response.Success(c, response.MsgTaskReportSubmittedSuccessfully)
}
