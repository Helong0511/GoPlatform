package admin

import (
	"fmt"
	"unicode/utf8"

	"ad-platform/internal/common"
)

// ValidatePlainPassword 校验明文密码长度（字符数下限、bcrypt 72 字节上限）。
func ValidatePlainPassword(pw string, minLen int) error {
	if minLen < 6 {
		minLen = 6
	}
	if utf8.RuneCountInString(pw) < minLen {
		return fmt.Errorf("%w: password shorter than %d characters", common.ErrWeakPassword, minLen)
	}
	if len(pw) > 72 {
		return fmt.Errorf("%w: password exceeds 72 bytes", common.ErrWeakPassword)
	}
	return nil
}
