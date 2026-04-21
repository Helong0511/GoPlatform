package ad

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"ad-platform/internal/common"
	"ad-platform/pkg/db"
	"ad-platform/pkg/redis"

	"gorm.io/gorm"
)

// AdTask 广告任务模型
type AdTask struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	Name                string    `json:"name" gorm:"size:100;not null"`
	Description         string    `json:"description" gorm:"type:text"`
	ClientID            *uint     `json:"client_id" gorm:"index"`
	Status              string    `json:"status" gorm:"size:20;default:'active';index"`
	Priority            int       `json:"priority" gorm:"default:0;index"`
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time"`
	TargetAudience      string    `json:"target_audience" gorm:"type:jsonb"`
	Creative            string    `json:"creative" gorm:"type:jsonb"`
	Budget              float64   `json:"budget" gorm:"type:decimal(10,2)"`
	ProxyStrategy       int       `json:"proxy_strategy" gorm:"default:1"`
	BizType             int       `json:"biz_type" gorm:"default:1"`
	Profile             string    `json:"profile" gorm:"size:50;default:'mobile'"`
	Plat                string    `json:"plat" gorm:"size:50;default:'googlead';index"`
	Cau                 int       `json:"cau" gorm:"default:0"`
	Os                  int       `json:"os" gorm:"default:1;index"`
	InterceptInfo       string    `json:"intercept_info" gorm:"type:jsonb"`
	Landurl             string    `json:"landurl" gorm:"type:text"`
	AllowCounty         string    `json:"allow_county" gorm:"type:jsonb"`
	FinalDomains        string    `json:"final_domains" gorm:"type:jsonb"`
	Info                string    `json:"info" gorm:"type:jsonb"`
	UserQuality         string    `json:"user_quality" gorm:"type:jsonb"`
	ClickHomeOffsetX    int       `json:"click_home_offset_x" gorm:"default:100"`
	ClickHomeOffsetY    int       `json:"click_home_offset_y" gorm:"default:100"`
	ClickResultOffsetX  int       `json:"click_result_offset_x" gorm:"default:100"`
	ClickResultOffsetY  int       `json:"click_result_offset_y" gorm:"default:100"`
	FKeyApi             string    `json:"f_key_api" gorm:"type:text"`
	PlanID              *int      `json:"plan_id" gorm:"index"`
	CreatedAt           time.Time `json:"created_at" gorm:"index"`
	UpdatedAt           time.Time `json:"updated_at"`
	LastAccessedAt      time.Time `json:"last_accessed_at" gorm:"default:CURRENT_TIMESTAMP;index"`
	AccessCount         int       `json:"access_count" gorm:"default:0;index"`
}

// AdService 广告服务
type AdService struct {
	db         *db.PostgresDB
	redis      *redis.RedisClient
	localCache map[string]*AdTask
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewAdService 创建广告服务
func NewAdService(db *db.PostgresDB, redis *redis.RedisClient) *AdService {
	return &AdService{
		db:         db,
		redis:      redis,
		localCache: make(map[string]*AdTask),
		cacheTTL:   5 * time.Minute,
	}
}

// GetAdTaskByID 根据ID获取广告任务
func (s *AdService) GetAdTaskByID(id uint) (*AdTask, error) {
	// 先从本地缓存获取
	cacheKey := s.getCacheKey(id)
	s.cacheMutex.RLock()
	if ad, ok := s.localCache[cacheKey]; ok {
		s.cacheMutex.RUnlock()
		return ad, nil
	}
	s.cacheMutex.RUnlock()

	// 再从Redis缓存获取
	cachedAd, err := s.redis.Get(cacheKey)
	if err == nil {
		var ad AdTask
		if err := json.Unmarshal([]byte(cachedAd), &ad); err == nil {
			// 更新本地缓存
			s.cacheMutex.Lock()
			s.localCache[cacheKey] = &ad
			s.cacheMutex.Unlock()
			return &ad, nil
		}
	}

	// 从数据库获取
	var ad AdTask
	err = s.db.DB.First(&ad, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrAdTaskNotFound
		}
		return nil, err
	}

	// 更新缓存
	s.updateCache(&ad)

	return &ad, nil
}

// ListAdTasks 列出所有广告任务
func (s *AdService) ListAdTasks(page, pageSize int) ([]AdTask, int64, error) {
	var ads []AdTask
	var total int64

	// 计算总数
	if err := s.db.DB.Model(&AdTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.DB.Order("priority desc, created_at desc").Offset(offset).Limit(pageSize).Find(&ads).Error
	if err != nil {
		return nil, 0, err
	}

	// 更新缓存
	for i := range ads {
		s.updateCache(&ads[i])
	}

	return ads, total, nil
}

// CreateAdTask 创建广告任务
func (s *AdService) CreateAdTask(ad *AdTask) error {
	err := s.db.DB.Create(ad).Error
	if err != nil {
		return err
	}

	// 更新缓存
	s.updateCache(ad)

	return nil
}

// UpdateAdTask 更新广告任务
func (s *AdService) UpdateAdTask(ad *AdTask) error {
	err := s.db.DB.Save(ad).Error
	if err != nil {
		return err
	}

	// 更新缓存
	s.updateCache(ad)

	return nil
}

// DeleteAdTask 删除广告任务
func (s *AdService) DeleteAdTask(id uint) error {
	err := s.db.DB.Delete(&AdTask{}, id).Error
	if err != nil {
		return err
	}

	// 删除缓存
	cacheKey := s.getCacheKey(id)
	s.redis.Delete(cacheKey)
	s.cacheMutex.Lock()
	delete(s.localCache, cacheKey)
	s.cacheMutex.Unlock()

	return nil
}

// GetActiveAdTasks 获取活跃的广告任务
func (s *AdService) GetActiveAdTasks() ([]AdTask, error) {
	var ads []AdTask

	// 从数据库获取活跃的广告任务，使用 UTC 标准时间
	err := s.db.DB.Where("status = ? AND start_time <= ? AND end_time >= ?", "active", time.Now().UTC(), time.Now().UTC()).Order("priority desc").Find(&ads).Error
	if err != nil {
		return nil, err
	}

	// 更新缓存
	for i := range ads {
		s.updateCache(&ads[i])
	}

	return ads, nil
}

// ListActiveAdTasks 列出活跃的广告任务（支持分页、筛选和排序）
func (s *AdService) ListActiveAdTasks(page, pageSize int, filters map[string]interface{}, sortField, sortOrder string) ([]AdTask, int64, error) {
	var ads []AdTask
	var total int64

	// 构建查询
	query := s.db.DB.Model(&AdTask{}).Where("status = ? AND start_time <= ? AND end_time >= ?", "active", time.Now().UTC(), time.Now().UTC())

	// 应用筛选条件
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("start_time >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("end_time <= ?", endTime)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 应用排序
	if sortField != "" {
		order := sortField
		if sortOrder != "" {
			order += " " + sortOrder
		} else {
			order += " desc"
		}
		query = query.Order(order)
	} else {
		// 默认排序
		query = query.Order("priority desc, created_at desc")
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&ads).Error
	if err != nil {
		return nil, 0, err
	}

	// 更新缓存
	for i := range ads {
		s.updateCache(&ads[i])
	}

	return ads, total, nil
}

// updateCache 更新缓存
func (s *AdService) updateCache(ad *AdTask) {
	if ad == nil {
		return
	}

	cacheKey := s.getCacheKey(ad.ID)

	// 更新Redis缓存
	adJSON, err := json.Marshal(ad)
	if err == nil {
		s.redis.Set(cacheKey, string(adJSON), s.cacheTTL)
	}

	// 更新本地缓存
	s.cacheMutex.Lock()
	s.localCache[cacheKey] = ad
	s.cacheMutex.Unlock()
}

// getCacheKey 获取缓存键
func (s *AdService) getCacheKey(id uint) string {
	return fmt.Sprintf("ad:task:%d", id)
}
