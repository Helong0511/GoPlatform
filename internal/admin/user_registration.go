package admin

import (
	"crypto/sha256"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-platform/internal/common"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RegistrationStatusPending  = "pending"
	RegistrationStatusApproved = "approved"
	RegistrationStatusRejected = "rejected"
)

type UserRegistration struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	Username   string     `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email      string     `json:"email" gorm:"uniqueIndex;size:100;not null"`
	Status     string     `json:"status" gorm:"size:20;not null;default:'pending'"`
	Reason     string     `json:"reason" gorm:"size:255"`
	ReviewedBy *uint      `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type UserRegistrationAudit struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	RegistrationID uint      `json:"registration_id" gorm:"index;not null"`
	OperatorID     uint      `json:"operator_id" gorm:"not null"`
	Action         string    `json:"action" gorm:"size:20;not null"`
	Reason         string    `json:"reason" gorm:"size:255"`
	CreatedAt      time.Time `json:"created_at"`
}

type RegistrationReviewResult struct {
	RegistrationID uint   `json:"registration_id"`
	Approved       bool   `json:"approved"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password,omitempty"`
	APIKey         string `json:"api_key,omitempty"`
}

type UserAPIKey struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"index;not null"`
	KeyHash    string     `json:"-" gorm:"size:64;uniqueIndex;not null"`
	Name       string     `json:"name" gorm:"size:50;default:'default'"`
	Scope      string     `json:"scope" gorm:"size:200;default:'all'"`
	Status     bool       `json:"status" gorm:"default:true"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP string     `json:"last_used_ip" gorm:"size:50"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (s *UserService) CreateRegistration(username, email string) (*UserRegistration, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))
	if username == "" || email == "" {
		return nil, errors.New("username and email are required")
	}

	reg := &UserRegistration{
		Username: username,
		Email:    email,
		Status:   RegistrationStatusPending,
	}
	if err := s.db.DB.Create(reg).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("%w: username or email already exists", common.ErrDuplicateEntry)
		}
		return nil, err
	}
	return reg, nil
}

func (s *UserService) ListPendingRegistrations(page, pageSize int) ([]UserRegistration, int64, error) {
	var regs []UserRegistration
	var total int64

	if err := s.db.DB.Model(&UserRegistration{}).
		Where("status = ?", RegistrationStatusPending).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := s.db.DB.Where("status = ?", RegistrationStatusPending).
		Order("created_at ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&regs).Error; err != nil {
		return nil, 0, err
	}

	return regs, total, nil
}

func (s *UserService) CountPendingRegistrations() (int64, error) {
	var total int64
	err := s.db.DB.Model(&UserRegistration{}).
		Where("status = ?", RegistrationStatusPending).
		Count(&total).Error
	return total, err
}

func (s *UserService) ReviewRegistration(registrationID, operatorID uint, approved bool, reason string) (*RegistrationReviewResult, error) {
	reason = strings.TrimSpace(reason)
	action := RegistrationStatusRejected
	if approved {
		action = RegistrationStatusApproved
	}

	result := &RegistrationReviewResult{}
	err := s.db.DB.Transaction(func(tx *gorm.DB) error {
		var reg UserRegistration
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&reg, registrationID).Error; err != nil {
			return err
		}
		if reg.Status != RegistrationStatusPending {
			return fmt.Errorf("registration already reviewed")
		}

		now := time.Now().UTC()
		reg.Status = action
		reg.Reason = reason
		reg.ReviewedBy = &operatorID
		reg.ReviewedAt = &now
		if err := tx.Save(&reg).Error; err != nil {
			return err
		}

		if err := tx.Create(&UserRegistrationAudit{
			RegistrationID: reg.ID,
			OperatorID:     operatorID,
			Action:         action,
			Reason:         reason,
		}).Error; err != nil {
			return err
		}

		result.RegistrationID = reg.ID
		result.Approved = approved
		result.Username = reg.Username
		result.Email = reg.Email

		if !approved {
			return nil
		}

		password, err := generateSecureToken(12)
		if err != nil {
			return err
		}
		apiKey, err := generateSecureToken(24)
		if err != nil {
			return err
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user := &User{
			Username: reg.Username,
			Email:    reg.Email,
			Password: string(hashedPassword),
			Status:   true,
		}
		if err := tx.Create(user).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fmt.Errorf("%w: user already exists", common.ErrDuplicateEntry)
			}
			return err
		}
		var customerRole Role
		if err := tx.Where("name = ?", "customer").First(&customerRole).Error; err != nil {
			return err
		}
		if err := tx.Model(user).Association("Roles").Append(&customerRole); err != nil {
			return err
		}
		apiKeyHash := hashAPIKey(apiKey)
		if err := tx.Create(&UserAPIKey{
			UserID:  user.ID,
			KeyHash: apiKeyHash,
			Name:    "default",
			Scope:   "task:read,report:write",
			Status:  true,
		}).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fmt.Errorf("%w: api key conflict", common.ErrDuplicateEntry)
			}
			return err
		}

		result.Password = password
		result.APIKey = apiKey
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *UserService) ListRegistrationAudits(registrationID uint) ([]UserRegistrationAudit, error) {
	var audits []UserRegistrationAudit
	err := s.db.DB.Where("registration_id = ?", registrationID).
		Order("created_at DESC").
		Find(&audits).Error
	return audits, err
}

func generateSecureToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
