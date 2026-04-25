package common

import (
	"errors"

	"gorm.io/gorm"
)

var (
	// Not found domain errors (use errors.Is for matching).
	ErrUserNotFound       = errors.New("user not found")
	ErrClientNotFound     = errors.New("client not found")
	ErrAdTaskNotFound     = errors.New("ad task not found")
	ErrRoleNotFound       = errors.New("role not found")
	ErrDepartmentNotFound = errors.New("department not found")
	ErrAPIKeyNotFound     = errors.New("api key not found")
	ErrDuplicateEntry     = errors.New("duplicate entry")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidArgument    = errors.New("invalid argument")
)

func IsDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDuplicateEntry) || errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return false
}
