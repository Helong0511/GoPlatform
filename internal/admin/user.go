package admin

import (
	"errors"
	"time"

	"ad-platform/internal/common"
	"ad-platform/pkg/db"
	"ad-platform/pkg/redis"
	"ad-platform/pkg/util"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint        `json:"id" gorm:"primaryKey"`
	Username     string      `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Password     string      `json:"-" gorm:"size:100;not null"`
	Email        string      `json:"email" gorm:"uniqueIndex;size:100;not null"`
	Name         string      `json:"name" gorm:"size:100"`
	Phone        string      `json:"phone" gorm:"size:20"`
	DepartmentID *uint       `json:"department_id" gorm:"default:null"`
	Timezone     string      `json:"timezone" gorm:"size:50;default:'Asia/Shanghai'"`
	Status       bool        `json:"status" gorm:"default:true"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Roles        []Role      `json:"roles" gorm:"many2many:user_roles;"`
	Department   *Department `json:"department,omitempty" gorm:"foreignKey:DepartmentID"`
}

// UserService 用户服务
type UserService struct {
	db             *db.PostgresDB
	redis          *redis.RedisClient
	userLogService *UserLogService
}

// NewUserService 创建用户服务
func NewUserService(db *db.PostgresDB, redis *redis.RedisClient) *UserService {
	userLogService := NewUserLogService(db)
	return &UserService{
		db:             db,
		redis:          redis,
		userLogService: userLogService,
	}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(user *User) error {
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	// 创建用户
	return s.db.DB.Create(user).Error
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id uint) (*User, error) {
	var user User
	err := s.db.DB.Preload("Roles.Permissions").Preload("Department").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*User, error) {
	var user User
	err := s.db.DB.Preload("Roles.Permissions").Preload("Department").Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(user *User) error {
	// 如果密码不为空，加密密码
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.Password = string(hashedPassword)
	}

	// 更新用户
	return s.db.DB.Save(user).Error
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id uint) error {
	return s.db.DB.Delete(&User{}, id).Error
}

// ListUsers 列出所有用户
func (s *UserService) ListUsers(page, pageSize int) ([]User, int64, error) {
	var users []User
	var total int64
	page, pageSize = util.NormalizePagination(page, pageSize)

	// 计算总数
	if err := s.db.DB.Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.DB.Preload("Roles").Preload("Department").Offset(offset).Limit(pageSize).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// EnableUser 启用用户
func (s *UserService) EnableUser(id, operatorID uint, ip string) error {
	var user User
	err := s.db.DB.First(&user, id).Error
	if err != nil {
		return err
	}

	oldStatus := user.Status

	err = s.db.DB.Model(&User{}).Where("id = ?", id).Update("status", true).Error
	if err != nil {
		return err
	}

	return s.userLogService.LogUserStatusChange(id, operatorID, oldStatus, true, ip)
}

// DisableUser 禁用用户
func (s *UserService) DisableUser(id, operatorID uint, ip string) error {
	var user User
	err := s.db.DB.First(&user, id).Error
	if err != nil {
		return err
	}

	oldStatus := user.Status

	err = s.db.DB.Model(&User{}).Where("id = ?", id).Update("status", false).Error
	if err != nil {
		return err
	}

	return s.userLogService.LogUserStatusChange(id, operatorID, oldStatus, false, ip)
}

// GetUserStatusLogs 获取用户状态变更日志
func (s *UserService) GetUserStatusLogs(userID uint, page, pageSize int) ([]UserStatusLog, int64, error) {
	return s.userLogService.GetUserStatusLogs(userID, page, pageSize)
}

// CheckPassword 检查密码
func (s *UserService) CheckPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

// Client 客户模型（客户=下游=渠道=商户）
type Client struct {
	ID            uint        `json:"id" gorm:"primaryKey"`
	Name          string      `json:"name" gorm:"size:100;not null"`
	Code          string      `json:"code" gorm:"uniqueIndex;size:50;not null"`
	Contact       string      `json:"contact" gorm:"size:100"`
	Phone         string      `json:"phone" gorm:"size:20"`
	Email         string      `json:"email" gorm:"size:100"`
	DepartmentID  *uint       `json:"department_id" gorm:"default:null"`
	SalespersonID *uint       `json:"salesperson_id" gorm:"default:null"`
	Status        bool        `json:"status" gorm:"default:true"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	Department    *Department `json:"department,omitempty" gorm:"foreignKey:DepartmentID"`
	Salesperson   *User       `json:"salesperson,omitempty" gorm:"foreignKey:SalespersonID"`
}

// CreateClient 创建客户
func (s *UserService) CreateClient(client *Client) error {
	return s.db.DB.Create(client).Error
}

// GetClientByID 根据ID获取客户
func (s *UserService) GetClientByID(id uint) (*Client, error) {
	var client Client
	err := s.db.DB.Preload("Department").Preload("Salesperson").First(&client, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

// UpdateClient 更新客户
func (s *UserService) UpdateClient(client *Client) error {
	return s.db.DB.Save(client).Error
}

// DeleteClient 删除客户
func (s *UserService) DeleteClient(id uint) error {
	return s.db.DB.Delete(&Client{}, id).Error
}

// ListClients 列出所有客户
func (s *UserService) ListClients(page, pageSize int) ([]Client, int64, error) {
	var clients []Client
	var total int64
	page, pageSize = util.NormalizePagination(page, pageSize)

	// 计算总数
	if err := s.db.DB.Model(&Client{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.DB.Preload("Department").Preload("Salesperson").Offset(offset).Limit(pageSize).Find(&clients).Error
	if err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}
