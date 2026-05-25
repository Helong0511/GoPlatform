package admin

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"ad-platform/internal/common"

	"gorm.io/gorm"
)

func (s *UserService) ListUserAPIKeys(userID uint) ([]UserAPIKey, error) {
	var keys []UserAPIKey
	err := s.db.DB.Where("user_id = ?", userID).Order("id DESC").Find(&keys).Error
	return keys, err
}

func (s *UserService) DisableUserAPIKey(userID, keyID uint) error {
	result := s.db.DB.Model(&UserAPIKey{}).
		Where("id = ? AND user_id = ?", keyID, userID).
		Update("status", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.ErrAPIKeyNotFound
	}
	return nil
}

func (s *UserService) RotateUserAPIKey(userID uint, name, scope string, expiresAt *time.Time) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	normalizedScope, ok := normalizeAndValidateScope(scope)
	if !ok {
		return "", fmt.Errorf("%w: invalid api key scope", common.ErrInvalidArgument)
	}
	rawKey, err := generateSecureToken(24)
	if err != nil {
		return "", err
	}
	keyHash := hashAPIKey(rawKey)

	err = s.db.DB.Transaction(func(tx *gorm.DB) error {
		// 保留历史，统一失效当前启用密钥
		if err := tx.Model(&UserAPIKey{}).
			Where("user_id = ? AND status = ?", userID, true).
			Update("status", false).Error; err != nil {
			return err
		}
		if err := tx.Create(&UserAPIKey{
			UserID:    userID,
			KeyHash:   keyHash,
			Name:      name,
			Scope:     normalizedScope,
			Status:    true,
			ExpiresAt: expiresAt,
		}).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fmt.Errorf("%w: api key conflict", common.ErrDuplicateEntry)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return rawKey, nil
}

func (s *UserService) touchAPIKeyLastUsed(id uint, ip string) error {
	now := time.Now().UTC()
	return s.db.DB.Model(&UserAPIKey{}).Where("id = ?", id).
		Updates(map[string]interface{}{"last_used_at": &now, "last_used_ip": strings.TrimSpace(ip)}).Error
}

func (s *UserService) ValidateAPIKey(apiKey, requiredScope, ip string) (*User, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, common.ErrUserNotFound
	}
	keyHash := hashAPIKey(apiKey)

	var key UserAPIKey
	now := time.Now().UTC()
	if err := s.db.DB.Where("key_hash = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", keyHash, true, now).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}
	if !scopeAllowed(key.Scope, requiredScope) {
		return nil, common.ErrForbidden
	}
	keyID, clientIP := key.ID, ip
	go func() {
		_ = s.touchAPIKeyLastUsed(keyID, clientIP)
	}()
	return s.GetUserByID(key.UserID)
}

func scopeAllowed(granted, required string) bool {
	granted = strings.TrimSpace(strings.ToLower(granted))
	required = strings.TrimSpace(strings.ToLower(required))
	if required == "" {
		return true
	}
	if granted == "all" {
		return true
	}
	if granted == "" {
		return false
	}
	for _, item := range strings.Split(granted, ",") {
		if strings.TrimSpace(strings.ToLower(item)) == required {
			return true
		}
	}
	return false
}

func normalizeAndValidateScope(scope string) (string, bool) {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope == "" || scope == "all" {
		return "all", true
	}
	allowed := map[string]struct{}{
		"report:write": {},
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, 2)
	for _, item := range strings.Split(scope, ",") {
		token := strings.TrimSpace(strings.ToLower(item))
		if token == "" {
			continue
		}
		if _, ok := allowed[token]; !ok {
			return "", false
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		normalized = append(normalized, token)
	}
	if len(normalized) == 0 {
		return "all", true
	}
	slices.Sort(normalized)
	return strings.Join(normalized, ","), true
}
