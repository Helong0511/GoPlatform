package tests

import (
	"strings"
	"testing"

	"ad-platform/pkg/db"
)

func TestMySQLTLSQueryParamMapping(t *testing.T) {
	cases := []struct {
		sslmode string
		want    string
	}{
		{"disable", "tls=false"},
		{"", "tls=false"},
		{"true", "tls=true"},
		{"require", "tls=true"},
		{"skip-verify", "tls=skip-verify"},
		{"preferred", "tls=preferred"},
	}
	for _, tc := range cases {
		got := db.MySQLTLSQueryParam(tc.sslmode)
		if got != tc.want {
			t.Fatalf("sslmode %q: want %q, got %q", tc.sslmode, tc.want, got)
		}
		if !strings.Contains(got, "tls=") {
			t.Fatalf("expected tls= in param, got %q", got)
		}
	}
}
