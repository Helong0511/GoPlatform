package admin

import (
	"errors"
	"time"

	"ad-platform/internal/common"
	"ad-platform/pkg/db"

	"gorm.io/gorm"
)

// Role 角色模型
type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"uniqueIndex;size:50;not null"`
	Description string       `json:"description" gorm:"size:200"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `json:"permissions" gorm:"many2many:role_permissions;"`
}

// Permission 权限模型
type Permission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;size:50;not null"`
	Description string    `json:"description" gorm:"size:200"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleService 角色服务
type RoleService struct {
	db *db.PostgresDB
}

// NewRoleService 创建角色服务
func NewRoleService(db *db.PostgresDB) *RoleService {
	return &RoleService{
		db: db,
	}
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(role *Role) error {
	return s.db.DB.Create(role).Error
}

// GetRoleByID 根据ID获取角色
func (s *RoleService) GetRoleByID(id uint) (*Role, error) {
	var role Role
	err := s.db.DB.Preload("Permissions").First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(role *Role) error {
	return s.db.DB.Save(role).Error
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(id uint) error {
	return s.db.DB.Delete(&Role{}, id).Error
}

// ListRoles 列出所有角色
func (s *RoleService) ListRoles() ([]Role, error) {
	var roles []Role
	err := s.db.DB.Preload("Permissions").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// AssignPermissions 为角色分配权限
func (s *RoleService) AssignPermissions(roleID uint, permissionIDs []uint) error {
	// 开始事务
	return s.db.DB.Transaction(func(tx *gorm.DB) error {
		// 先清除角色的所有权限
		if err := tx.Model(&Role{ID: roleID}).Association("Permissions").Clear(); err != nil {
			return err
		}

		// 分配新权限
		var permissions []Permission
		if err := tx.Find(&permissions, permissionIDs).Error; err != nil {
			return err
		}

		if err := tx.Model(&Role{ID: roleID}).Association("Permissions").Append(permissions); err != nil {
			return err
		}

		return nil
	})
}

// AssignRoles 为用户分配角色
func (s *RoleService) AssignRoles(userID uint, roleIDs []uint) error {
	// 开始事务
	return s.db.DB.Transaction(func(tx *gorm.DB) error {
		// 先清除用户的所有角色
		if err := tx.Model(&User{ID: userID}).Association("Roles").Clear(); err != nil {
			return err
		}

		// 分配新角色
		var roles []Role
		if err := tx.Find(&roles, roleIDs).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{ID: userID}).Association("Roles").Append(roles); err != nil {
			return err
		}

		return nil
	})
}

// ListPermissions 列出所有权限
func (s *RoleService) ListPermissions() ([]Permission, error) {
	var permissions []Permission
	err := s.db.DB.Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}
