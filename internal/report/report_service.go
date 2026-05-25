package report

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"ad-platform/pkg/db"
	"ad-platform/pkg/logger"
	"ad-platform/pkg/pulsar"
	"ad-platform/pkg/util"
)

// AdReport 广告上报模型
type AdReport struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AdID          uint      `json:"ad_id" gorm:"index"`
	UserID        string    `json:"user_id" gorm:"size:50"`
	Action        string    `json:"action" gorm:"size:20;index;not null"`
	DeviceInfo    string    `json:"device_info" gorm:"type:jsonb"`
	IP            string    `json:"ip" gorm:"size:50"`
	ClientInner   string    `json:"client_inner" gorm:"size:50"`
	ClientOuter   string    `json:"client_outer" gorm:"size:50"`
	RealIP        string    `json:"real_ip" gorm:"size:50"`
	Click         int       `json:"click" gorm:"default:0"`
	UseTime       float64   `json:"use_time" gorm:"type:decimal(10,3)"`
	Ads           int       `json:"ads" gorm:"default:0"`
	HomePage      bool      `json:"home_page" gorm:"default:false"`
	SubpageAccess int       `json:"subpage_access" gorm:"default:0"`
	Referer       string    `json:"referer" gorm:"type:text"`
	Country       string    `json:"country" gorm:"size:10"`
	Seq           string    `json:"seq" gorm:"size:50"`
	Error         string    `json:"error" gorm:"type:text"`
	ShouldClick   bool      `json:"should_click" gorm:"default:false"`
	ShouldSubpage bool      `json:"should_subpage" gorm:"default:false"`
	IOS           bool      `json:"ios" gorm:"default:false"`
	ClickAble     bool      `json:"click_able" gorm:"default:false"`
	SourceType    string    `json:"source_type" gorm:"size:50"`
	IPSource      int       `json:"ip_source" gorm:"default:0"`
	AdType        int       `json:"ad_type" gorm:"default:0"`
	IsReview      int       `json:"is_review" gorm:"default:0"`
	LastSeq       string    `json:"last_seq" gorm:"size:50"`
	ScreenWidth   int       `json:"screen_width" gorm:"default:0"`
	RealURL       string    `json:"real_url" gorm:"type:text"`
	OriginURL     string    `json:"origin_url" gorm:"type:text"`
	IsSave        int       `json:"is_save" gorm:"default:0"`
	GPUError      int       `json:"gpu_error" gorm:"default:0"`
	ReqAdsCount   int       `json:"req_ads_count" gorm:"default:0"`
	MatchCount    int       `json:"match_count" gorm:"default:0"`
	ActiveView    int       `json:"active_view" gorm:"default:0"`
	SendBytes     string    `json:"send_bytes" gorm:"size:50;default:'0'"`
	RecvBytes     string    `json:"recv_bytes" gorm:"size:50;default:'0'"`
	SleepInfo     string    `json:"sleep_info" gorm:"type:text"`
	StartTime     int64     `json:"start_time"`
	EndTime       int64     `json:"end_time"`
	PlanID        int       `json:"plan_id"`
	Time          string    `json:"time" gorm:"size:20"`
	Timestamp     time.Time `json:"timestamp" gorm:"index;not null"`
}

// ReportService 上报服务
type ReportService struct {
	db             *db.PostgresDB
	pulsarProducer *pulsar.PulsarProducer
	batchSize      int
	batch          []AdReport
	batchMutex     sync.Mutex
	ticker         *time.Ticker
	done           chan struct{}
	closeOnce      sync.Once
}

var errReportDBNotInitialized = errors.New("report database is not initialized")
var reportLogger = logger.NewLogger("info")

// NewReportService 创建上报服务
func NewReportService(db *db.PostgresDB, pulsarProducer *pulsar.PulsarProducer) *ReportService {
	service := &ReportService{
		db:             db,
		pulsarProducer: pulsarProducer,
		batchSize:      1, // 减少批次大小，确保数据及时处理
		batch:          make([]AdReport, 0, 100),
		done:           make(chan struct{}),
	}

	// 启动定时批处理
	service.ticker = time.NewTicker(1 * time.Second) // 减少处理间隔
	go service.processBatch()

	return service
}

// ReportAd 上报广告执行数据
func (s *ReportService) ReportAd(report *AdReport) error {
	// 如果Pulsar生产者可用，则发送到Pulsar
	if s.pulsarProducer != nil {
		reportJSON, err := json.Marshal(report)
		if err != nil {
			return err
		}

		err = s.pulsarProducer.Produce(reportJSON)
		if err == nil {
			return nil
		}
		// 如果Pulsar发送失败，降级为直接写入数据库
		reportLogger.Warn("Pulsar produce failed, fallback to DB write: %v", err)
	}

	// Pulsar不可用时，直接添加到批次写入数据库
	s.batchMutex.Lock()
	s.batch = append(s.batch, *report)
	batchLen := len(s.batch)
	s.batchMutex.Unlock()

	// 如果批次达到阈值，立即处理
	if batchLen >= s.batchSize {
		s.batchInsert()
	}

	return nil
}

// processBatch 处理批量数据
func (s *ReportService) processBatch() {
	for {
		select {
		case <-s.ticker.C:
			s.batchMutex.Lock()
			hasData := len(s.batch) > 0
			s.batchMutex.Unlock()

			if hasData {
				// 批量写入数据库
				s.batchInsert()
			}
		case <-s.done:
			return
		}
	}
}

// batchInsert 批量插入数据
func (s *ReportService) batchInsert() {
	// 批量插入（在锁外执行）
	if s.db == nil || s.db.DB == nil {
		reportLogger.Warn("Skip batch insert: database is not initialized")
		return
	}
	s.batchMutex.Lock()
	// 检查批次是否为空
	if len(s.batch) == 0 {
		s.batchMutex.Unlock()
		return
	}

	// 创建批次的副本，避免在数据库操作期间持有锁
	batchCopy := make([]AdReport, len(s.batch))
	copy(batchCopy, s.batch)

	// 清空批次
	s.batch = make([]AdReport, 0, s.batchSize)
	s.batchMutex.Unlock()

	err := s.db.DB.Create(&batchCopy).Error
	if err != nil {
		reportLogger.Error("Failed to batch insert: %v", err)
		s.batchMutex.Lock()
		s.batch = append(batchCopy, s.batch...)
		s.batchMutex.Unlock()
		return
	}

	reportLogger.Info("Successfully inserted %d reports", len(batchCopy))
}

const maxPulsarPoisonRedeliveries uint32 = 5

// ConsumeReports 消费 Pulsar 消息并同步写入数据库；落库成功后 Ack；失败 Nack 重试；毒消息超限后 Ack 丢弃。
func (s *ReportService) ConsumeReports(pulsarConsumer *pulsar.PulsarConsumer) {
	for {
		msg, err := pulsarConsumer.Consume()
		if err != nil {
			reportLogger.Error("Failed to consume message: %v", err)
			continue
		}

		var report AdReport
		if err := json.Unmarshal(msg.Payload(), &report); err != nil {
			if msg.RedeliveryCount() >= maxPulsarPoisonRedeliveries {
				reportLogger.Error("dropping poison pulsar message after %d redeliveries: %v", msg.RedeliveryCount(), err)
				pulsarConsumer.Consumer.Ack(msg)
			} else {
				reportLogger.Error("failed to unmarshal pulsar message (nack): %v", err)
				pulsarConsumer.Consumer.Nack(msg)
			}
			continue
		}

		if err := s.persistReportFromQueue(&report); err != nil {
			reportLogger.Error("Failed to persist report from queue: %v", err)
			pulsarConsumer.Consumer.Nack(msg)
			continue
		}

		pulsarConsumer.Consumer.Ack(msg)
	}
}

func (s *ReportService) persistReportFromQueue(report *AdReport) error {
	if err := s.ensureDBReady(); err != nil {
		return err
	}
	return s.db.DB.Create(report).Error
}

// Close 关闭服务
func (s *ReportService) Close() {
	s.closeOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		if s.done != nil {
			close(s.done)
		}

		// 处理剩余批次，避免锁重入死锁。
		s.batchInsert()
	})
}

// GetTotalReports 获取总上报数
func (s *ReportService) GetTotalReports() (int64, error) {
	var count int64
	if err := s.ensureDBReady(); err != nil {
		return 0, err
	}

	// 查询总上报数量
	err := s.db.DB.Model(&AdReport{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetTotalClicks 获取总点击数
func (s *ReportService) GetTotalClicks() (int64, error) {
	var totalClicks int64
	if err := s.ensureDBReady(); err != nil {
		return 0, err
	}

	// 查询总点击数量（统计Click字段的总和）
	err := s.db.DB.Model(&AdReport{}).Select("COALESCE(SUM(click), 0) as total_clicks").Row().Scan(&totalClicks)
	if err != nil {
		return 0, err
	}

	return totalClicks, nil
}

// GetReportStats 获取上报统计数据
func (s *ReportService) GetReportStats() (map[string]interface{}, error) {
	// 获取总上报数
	totalReports, err := s.GetTotalReports()
	if err != nil {
		return nil, err
	}

	// 获取总点击数
	totalClicks, err := s.GetTotalClicks()
	if err != nil {
		return nil, err
	}

	// report_rate：totalClicks/totalReports 折算为百分比（近似「每条上报对应点击量」的均值比例），并非漏斗意义上的「上报到达率」。
	reportRate := 0.0
	if totalReports > 0 {
		reportRate = float64(totalClicks) / float64(totalReports) * 100
	}

	// 获取今日上报数
	todayReports, err := s.GetTodayReports()
	if err != nil {
		return nil, err
	}

	// 构建统计数据
	stats := map[string]interface{}{
		"total_reports": totalReports,
		"total_clicks":  totalClicks,
		"report_rate":   reportRate,
		"today_reports": todayReports,
	}

	return stats, nil
}

// GetReportStatsByTimezone 根据时区获取上报统计数据
func (s *ReportService) GetReportStatsByTimezone(timezone string) (map[string]interface{}, error) {
	if err := s.ensureDBReady(); err != nil {
		return nil, err
	}

	// 合并查询，减少数据库交互次数
	var totalReports int64
	var totalClicks int64

	// 同时查询总上报数和总点击数
	err := s.db.DB.Model(&AdReport{}).Select("COUNT(*) as total_reports, COALESCE(SUM(click), 0) as total_clicks").Row().Scan(&totalReports, &totalClicks)
	if err != nil {
		reportLogger.Error("Failed to get report stats: %v", err)
		return nil, err
	}

	// report_rate：totalClicks/totalReports 折算为百分比，语义见 GetReportStats。
	reportRate := 0.0
	if totalReports > 0 {
		reportRate = float64(totalClicks) / float64(totalReports) * 100
	}

	// 获取今日上报数（根据时区）
	todayReports, err := s.GetTodayReportsByTimezone(timezone)
	if err != nil {
		reportLogger.Warn("Failed to get today reports: %v", err)
		// 如果获取今日上报数失败，不影响整体统计，使用默认值
		todayReports = 0
	}

	// 构建统计数据
	stats := map[string]interface{}{
		"total_reports": totalReports,
		"total_clicks":  totalClicks,
		"report_rate":   reportRate,
		"today_reports": todayReports,
	}

	return stats, nil
}

// GetTodayReports 获取今日上报数据
func (s *ReportService) GetTodayReports() (int64, error) {
	var count int64
	if err := s.ensureDBReady(); err != nil {
		return 0, err
	}

	// 获取今日开始时间（使用UTC时间）
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// 查询今日上报数量
	err := s.db.DB.Model(&AdReport{}).Where("timestamp >= ?", today).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetTodayReportsByTimezone 根据用户时区获取今日上报数据
func (s *ReportService) GetTodayReportsByTimezone(timezone string) (int64, error) {
	var count int64
	if err := s.ensureDBReady(); err != nil {
		return 0, err
	}

	// 根据时区获取今日开始时间和结束时间
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// 如果时区无效，使用 UTC 时区
		reportLogger.Warn("Invalid timezone %s, using UTC timezone: %v", timezone, err)
		loc = time.UTC
	}

	// 获取当地时间的今日开始时间和结束时间
	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)

	// 将当地时间转换为 UTC 时间
	todayStartUTC := todayStart.UTC()
	todayEndUTC := todayEnd.UTC()

	// 查询今日上报数量（使用 UTC 时间范围）
	err = s.db.DB.Model(&AdReport{}).Where("timestamp >= ? AND timestamp <= ?", todayStartUTC, todayEndUTC).Count(&count).Error
	if err != nil {
		reportLogger.Error("Failed to get today reports count: %v", err)
		return 0, err
	}

	return count, nil
}

// ListAdReports 获取广告上报列表
func (s *ReportService) ListAdReports(page, pageSize int) ([]AdReport, int64, error) {
	var reports []AdReport
	var total int64
	if err := s.ensureDBReady(); err != nil {
		return nil, 0, err
	}
	page, pageSize = util.NormalizePagination(page, pageSize)

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	err := s.db.DB.Model(&AdReport{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 查询数据
	err = s.db.DB.Order("timestamp desc").Offset(offset).Limit(pageSize).Find(&reports).Error
	if err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

// ListAdReportsWithFilter 根据条件筛选广告上报列表
func (s *ReportService) ListAdReportsWithFilter(page, pageSize int, filters map[string]interface{}) ([]AdReport, int64, error) {
	var reports []AdReport
	var total int64
	if err := s.ensureDBReady(); err != nil {
		return nil, 0, err
	}

	page, pageSize = util.NormalizePagination(page, pageSize)

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建查询
	query := s.db.DB.Model(&AdReport{})

	// 应用筛选条件
	if adID, ok := filters["ad_id"].(uint); ok && adID > 0 {
		query = query.Where("ad_id = ?", adID)
	}

	if action, ok := filters["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}

	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("timestamp >= ?", startTime)
	}

	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("timestamp <= ?", endTime)
	}

	// 查询总数
	err := query.Count(&total).Error
	if err != nil {
		reportLogger.Error("Failed to count reports: %v", err)
		return nil, 0, err
	}

	// 查询数据
	err = query.Order("timestamp desc").Offset(offset).Limit(pageSize).Find(&reports).Error
	if err != nil {
		reportLogger.Error("Failed to query reports: %v", err)
		return nil, 0, err
	}

	// 如果没有数据，返回空数组而不是 nil
	if reports == nil {
		reports = []AdReport{}
	}

	return reports, total, nil
}

func (s *ReportService) ensureDBReady() error {
	if s == nil || s.db == nil || s.db.DB == nil {
		return errReportDBNotInitialized
	}
	return nil
}
