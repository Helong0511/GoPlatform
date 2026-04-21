package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-platform/internal/security"
)

const testAESKeyBase64 = "awwYDrn0ojn2p7y2hNL/kg=="

func TestSecurityManagerJWTAndAESRoundTrip(t *testing.T) {
	manager, err := security.NewSecurityManager("")
	if err != nil {
		t.Fatalf("failed to create security manager: %v", err)
	}

	token, err := manager.GenerateJWT("u-1001", "admin", "jwt-secret", 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}

	valid, err := manager.ValidateJWT(token, "jwt-secret")
	if err != nil {
		t.Fatalf("failed to validate jwt: %v", err)
	}
	if !valid {
		t.Fatalf("expected jwt to be valid")
	}

	encrypted, err := manager.Encrypt("payload-data", testAESKeyBase64)
	if err != nil {
		t.Fatalf("failed to encrypt data: %v", err)
	}

	decrypted, err := manager.Decrypt(encrypted, testAESKeyBase64)
	if err != nil {
		t.Fatalf("failed to decrypt data: %v", err)
	}
	if decrypted != "payload-data" {
		t.Fatalf("expected decrypted payload-data, got %s", decrypted)
	}
}

func TestSecurityManagerRustLibDetection(t *testing.T) {
	manager, err := security.NewSecurityManager("")
	if err != nil {
		t.Fatalf("failed to create manager with empty path: %v", err)
	}
	if manager.IsRustLibEnabled() {
		t.Fatalf("expected rust lib disabled for empty path")
	}

	tempDir := t.TempDir()
	libPath := filepath.Join(tempDir, "rust_security.dll")
	if err := os.WriteFile(libPath, []byte("mock"), 0644); err != nil {
		t.Fatalf("failed to prepare temp rust lib file: %v", err)
	}

	managerWithLib, err := security.NewSecurityManager(libPath)
	if err != nil {
		t.Fatalf("failed to create manager with lib path: %v", err)
	}
	if !managerWithLib.IsRustLibEnabled() {
		t.Fatalf("expected rust lib enabled when file exists")
	}
}
