package admin

import (
	"errors"
	"time"

	"ad-platform/internal/common"
	"ad-platform/pkg/db"

	"gorm.io/gorm"
)

// Department 部门模型
type Department struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;size:100;not null"`
	Description string    `json:"description" gorm:"size:200"`
	ParentID    *uint     `json:"parent_id" gorm:"default:null"`
	Status      bool      `json:"status" gorm:"default:true"` // true: 正常, false: 归档
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Parent      *Department `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children    []Department `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Users       []User      `json:"users,omitempty" gorm:"foreignKey:DepartmentID"`
}

// DepartmentService 部门服务
type DepartmentService struct {
	db *db.PostgresDB
}

// NewDepartmentService 创建部门服务
func NewDepartmentService(db *db.PostgresDB) *DepartmentService {
	return &DepartmentService{
		db: db,
	}
}

// CreateDepartment 创建部门
func (s *DepartmentService) CreateDepartment(dept *Department) error {
	return s.db.DB.Create(dept).Error
}

// GetDepartmentByID 根据ID获取部门
func (s *DepartmentService) GetDepartmentByID(id uint) (*Department, error) {
	var dept Department
	err := s.db.DB.First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

// GetDepartmentWithChildren 获取部门及其子部门
func (s *DepartmentService) GetDepartmentWithChildren(id uint) (*Department, error) {
	var dept Department
	err := s.db.DB.Preload("Children").First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

// UpdateDepartment 更新部门
func (s *DepartmentService) UpdateDepartment(dept *Department) error {
	return s.db.DB.Save(dept).Error
}

// ArchiveDepartment 归档部门
func (s *DepartmentService) ArchiveDepartment(id uint) error {
	return s.db.DB.Model(&Department{}).Where("id = ?", id).Update("status", false).Error
}

// RestoreDepartment 恢复部门
func (s *DepartmentService) RestoreDepartment(id uint) error {
	return s.db.DB.Model(&Department{}).Where("id = ?", id).Update("status", true).Error
}

// ListDepartments 列出所有部门
func (s *DepartmentService) ListDepartments(onlyActive bool) ([]Department, error) {
	var depts []Department
	query := s.db.DB.Preload("Parent")
	if onlyActive {
		query = query.Where("status = ?", true)
	}
	err := query.Find(&depts).Error
	if err != nil {
		return nil, err
	}
	return depts, nil
}

// ListDepartmentsWithHierarchy 列出部门层级结构
func (s *DepartmentService) ListDepartmentsWithHierarchy() ([]Department, error) {
	var rootDepts []Department
	err := s.db.DB.Where("parent_id IS NULL AND status = ?", true).Preload("Children").Find(&rootDepts).Error
	if err != nil {
		return nil, err
	}
	return rootDepts, nil
}
