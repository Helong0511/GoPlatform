package ad

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ad-platform/internal/common"
	"ad-platform/pkg/db"
	"ad-platform/pkg/redis"

	"gorm.io/gorm"
)

// AdTask 广告任务模型
type AdTask struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	Name               string    `json:"name" gorm:"size:100;not null"`
	Description        string    `json:"description" gorm:"type:text"`
	ClientID           *uint     `json:"client_id" gorm:"index"`
	Status             string    `json:"status" gorm:"size:20;default:'active';index"`
	Priority           int       `json:"priority" gorm:"default:0;index"`
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
	TargetAudience     string    `json:"target_audience" gorm:"type:jsonb"`
	Creative           string    `json:"creative" gorm:"type:jsonb"`
	Budget             float64   `json:"budget" gorm:"type:decimal(10,2)"`
	ProxyStrategy      int       `json:"proxy_strategy" gorm:"default:1"`
	BizType            int       `json:"biz_type" gorm:"default:1"`
	Profile            string    `json:"profile" gorm:"size:50;default:'mobile'"`
	Platform           string    `json:"plat" gorm:"column:plat;size:50;default:'googlead';index"`
	CAU                int       `json:"cau" gorm:"column:cau;default:0"`
	OS                 int       `json:"os" gorm:"column:os;default:1;index"`
	InterceptInfo      string    `json:"intercept_info" gorm:"type:jsonb"`
	LandURL            string    `json:"landurl" gorm:"column:landurl;type:text"`
	AllowCounty        string    `json:"allow_county" gorm:"type:jsonb"`
	FinalDomains       string    `json:"final_domains" gorm:"type:jsonb"`
	Info               string    `json:"info" gorm:"type:jsonb"`
	UserQuality        string    `json:"user_quality" gorm:"type:jsonb"`
	ClickHomeOffsetX   int       `json:"click_home_offset_x" gorm:"default:100"`
	ClickHomeOffsetY   int       `json:"click_home_offset_y" gorm:"default:100"`
	ClickResultOffsetX int       `json:"click_result_offset_x" gorm:"default:100"`
	ClickResultOffsetY int       `json:"click_result_offset_y" gorm:"default:100"`
	FKeyAPI            string    `json:"f_key_api" gorm:"column:f_key_api;type:text"`
	PlanID             *int      `json:"plan_id" gorm:"index"`
	CreatedAt          time.Time `json:"created_at" gorm:"index"`
	UpdatedAt          time.Time `json:"updated_at"`
	LastAccessedAt     time.Time `json:"last_accessed_at" gorm:"default:CURRENT_TIMESTAMP;index"`
	AccessCount        int       `json:"access_count" gorm:"default:0;index"`
}

type adCacheEntry struct {
	task    *AdTask
	expires time.Time
}

// AdService 广告服务
type AdService struct {
	db         *db.PostgresDB
	redis      *redis.RedisClient
	localCache map[string]adCacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewAdService 创建广告服务
func NewAdService(db *db.PostgresDB, redis *redis.RedisClient) *AdService {
	return &AdService{
		db:         db,
		redis:      redis,
		localCache: make(map[string]adCacheEntry),
		cacheTTL:   5 * time.Minute,
	}
}

// GetAdTaskByID 根据ID获取广告任务
func (s *AdService) GetAdTaskByID(id uint) (*AdTask, error) {
	// 先从本地缓存获取
	cacheKey := s.getCacheKey(id)
	s.cacheMutex.RLock()
	if entry, ok := s.localCache[cacheKey]; ok && time.Now().Before(entry.expires) {
		s.cacheMutex.RUnlock()
		return entry.task, nil
	}
	s.cacheMutex.RUnlock()

	// 再从Redis缓存获取
	cachedAd, err := s.redis.Get(cacheKey)
	if err == nil {
		var ad AdTask
		if err := json.Unmarshal([]byte(cachedAd), &ad); err == nil {
			// 更新本地缓存
			s.cacheMutex.Lock()
			s.localCache[cacheKey] = adCacheEntry{task: &ad, expires: time.Now().Add(s.cacheTTL)}
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
	ad.ID = 0
	ad.AccessCount = 0
	if strings.TrimSpace(ad.Status) == "" {
		ad.Status = "active"
	}
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
	var existing AdTask
	if err := s.db.DB.First(&existing, ad.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrAdTaskNotFound
		}
		return err
	}
	ad.AccessCount = existing.AccessCount
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
	result := s.db.DB.Delete(&AdTask{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.ErrAdTaskNotFound
	}

	// 删除缓存
	cacheKey := s.getCacheKey(id)
	s.redis.Delete(cacheKey)
	s.cacheMutex.Lock()
	delete(s.localCache, cacheKey)
	s.cacheMutex.Unlock()

	return nil
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
	s.localCache[cacheKey] = adCacheEntry{task: ad, expires: time.Now().Add(s.cacheTTL)}
	s.cacheMutex.Unlock()
}

// getCacheKey 获取缓存键
func (s *AdService) getCacheKey(id uint) string {
	return fmt.Sprintf("ad:task:%d", id)
}
