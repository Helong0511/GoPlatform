package tests

import (
	"errors"
	"strings"
	"testing"

	"ad-platform/internal/admin"
	"ad-platform/internal/common"
)

func TestValidatePlainPasswordLength(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		err := admin.ValidatePlainPassword("short", 8)
		if err == nil || !errors.Is(err, common.ErrWeakPassword) {
			t.Fatalf("expected ErrWeakPassword, got %v", err)
		}
	})

	t.Run("ok at boundary", func(t *testing.T) {
		if err := admin.ValidatePlainPassword("12345678", 8); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("over 72 bytes", func(t *testing.T) {
		long := strings.Repeat("a", 73)
		err := admin.ValidatePlainPassword(long, 6)
		if err == nil || !errors.Is(err, common.ErrWeakPassword) {
			t.Fatalf("expected ErrWeakPassword for >72 bytes, got %v", err)
		}
	})
}
