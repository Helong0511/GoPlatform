package common

import "errors"

var (
	// Not found domain errors (use errors.Is for matching).
	ErrUserNotFound       = errors.New("user not found")
	ErrClientNotFound     = errors.New("client not found")
	ErrAdTaskNotFound     = errors.New("ad task not found")
	ErrRoleNotFound       = errors.New("role not found")
	ErrDepartmentNotFound = errors.New("department not found")
)
