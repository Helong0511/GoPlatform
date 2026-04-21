package tests

import (
	"testing"

	"ad-platform/pkg/util"
)

func TestResolveSecret(t *testing.T) {
	const envKey = "TEST_SECRET_KEY_E2E"

	t.Run("env has highest priority", func(t *testing.T) {
		t.Setenv(envKey, "env-secret")
		value, source, err := util.ResolveSecret(envKey, "config-secret")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if value != "env-secret" || source != util.SecretSourceEnv {
			t.Fatalf("unexpected resolve result value=%s source=%s", value, source)
		}
	})

	t.Run("fallback to config when env is empty", func(t *testing.T) {
		t.Setenv(envKey, "")
		value, source, err := util.ResolveSecret(envKey, "config-secret")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if value != "config-secret" || source != util.SecretSourceConfig {
			t.Fatalf("unexpected resolve result value=%s source=%s", value, source)
		}
	})

	t.Run("return error when both are missing", func(t *testing.T) {
		t.Setenv(envKey, "")
		_, _, err := util.ResolveSecret(envKey, "")
		if err == nil {
			t.Fatalf("expected error when env and config secret are both missing")
		}
	})
}
