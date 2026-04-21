package admin

import (
	"time"

	"ad-platform/pkg/db"
)

// UserStatusLog 用户状态变更日志模型
type UserStatusLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"index;not null"`
	OperatorID uint      `json:"operator_id" gorm:"index"`
	OldStatus  bool      `json:"old_status"`
	NewStatus  bool      `json:"new_status"`
	IP         string    `json:"ip" gorm:"size:50"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserLogService 用户日志服务
type UserLogService struct {
	db *db.PostgresDB
}

// NewUserLogService 创建用户日志服务
func NewUserLogService(db *db.PostgresDB) *UserLogService {
	return &UserLogService{db: db}
}

// LogUserStatusChange 记录用户状态变更
func (s *UserLogService) LogUserStatusChange(userID, operatorID uint, oldStatus, newStatus bool, ip string) error {
	log := &UserStatusLog{
		UserID:     userID,
		OperatorID: operatorID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		IP:         ip,
		CreatedAt:  time.Now().UTC(), // 使用 UTC 标准时间保存
	}
	return s.db.DB.Create(log).Error
}

// GetUserStatusLogs 获取用户的状态变更日志
func (s *UserLogService) GetUserStatusLogs(userID uint, page, pageSize int) ([]UserStatusLog, int64, error) {
	var logs []UserStatusLog
	var total int64

	// 计算总数
	s.db.DB.Model(&UserStatusLog{}).Where("user_id = ?", userID).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error

	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetRecentStatusLogs 获取最近的状态变更日志
func (s *UserLogService) GetRecentStatusLogs(limit int) ([]UserStatusLog, error) {
	var logs []UserStatusLog
	err := s.db.DB.Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	return logs, nil
}
