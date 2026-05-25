package tests

import (
	"testing"
	"time"

	pkgauth "ad-platform/pkg/auth"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRejectsWrongIssuerOrAudience(t *testing.T) {
	const secret = "jwt-iss-aud-secret-key-32chars!!"
	mw := pkgauth.NewAuthMiddleware(secret, "ad-platform", "ad-platform-api")

	tok, err := mw.GenerateJWT("42", "UTC", []string{"admin"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	if _, err := mw.ValidateJWT(tok); err != nil {
		t.Fatalf("valid token should pass: %v", err)
	}

	wrongIss := pkgauth.NewAuthMiddleware(secret, "other-issuer", "ad-platform-api")
	if _, err := wrongIss.ValidateJWT(tok); err == nil {
		t.Fatalf("expected rejection when issuer mismatches")
	}

	wrongAud := pkgauth.NewAuthMiddleware(secret, "ad-platform", "other-api")
	if _, err := wrongAud.ValidateJWT(tok); err == nil {
		t.Fatalf("expected rejection when audience mismatches")
	}
}

func TestJWTWithoutIssuerAudienceStillWorks(t *testing.T) {
	mw := pkgauth.NewAuthMiddleware("legacy-secret-no-iss-aud-32b", "", "")
	tok, err := mw.GenerateJWT("1", "Asia/Shanghai", nil, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	claims, err := mw.ValidateJWT(tok)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}
	if claims.Issuer != "" {
		t.Fatalf("expected empty issuer, got %q", claims.Issuer)
	}
}

func TestJWTClaimsIncludeIssuerAudience(t *testing.T) {
	mw := pkgauth.NewAuthMiddleware("claim-check-secret-32-characters", "iss-x", "aud-y")
	tok, err := mw.GenerateJWT("9", "UTC", []string{"r"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	parser := jwt.NewParser(jwt.WithIssuer("iss-x"), jwt.WithAudience("aud-y"))
	var raw pkgauth.Claims
	_, err = parser.ParseWithClaims(tok, &raw, func(t *jwt.Token) (interface{}, error) {
		return []byte("claim-check-secret-32-characters"), nil
	})
	if err != nil {
		t.Fatalf("parse claims: %v", err)
	}
	if raw.Issuer != "iss-x" {
		t.Fatalf("issuer: got %q", raw.Issuer)
	}
	if len(raw.Audience) != 1 || raw.Audience[0] != "aud-y" {
		t.Fatalf("audience: got %v", raw.Audience)
	}
}
