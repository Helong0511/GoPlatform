package tests

import (
	"testing"

	"ad-platform/pkg/util"
)

func TestValidateProductionSecrets(t *testing.T) {
	if err := util.ValidateProductionSecrets(
		"12345678901234567890123456789012",
		"awwYDrn0ojn2p7y2hNL/kg==",
	); err != nil {
		t.Fatalf("expected valid secrets, got %v", err)
	}

	if err := util.ValidateProductionSecrets("short", "awwYDrn0ojn2p7y2hNL/kg=="); err == nil {
		t.Fatalf("expected error for short jwt")
	}
	if err := util.ValidateProductionSecrets("12345678901234567890123456789012", "not-base64!!!"); err == nil {
		t.Fatalf("expected error for invalid aes base64")
	}
	if err := util.ValidateProductionSecrets("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "awwYDrn0ojn2p7y2hNL/kg=="); err == nil {
		t.Fatalf("expected error for weak jwt secret")
	}
}

func TestIsProductionEnvironment(t *testing.T) {
	if !util.IsProductionEnvironment("production") || !util.IsProductionEnvironment("prod") {
		t.Fatalf("expected production aliases recognized")
	}
	if util.IsProductionEnvironment("development") {
		t.Fatalf("development must not be production")
	}
}
