package tests

import (
	"encoding/json"
	"testing"

	"ad-platform/internal/admin"
)

func TestRegistrationReviewResultOmitsCredentialsFromJSON(t *testing.T) {
	result := &admin.RegistrationReviewResult{
		RegistrationID: 1,
		Approved:       true,
		Username:       "u1",
		Email:          "u1@example.com",
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"password", "api_key"} {
		if _, ok := m[key]; ok {
			t.Fatalf("JSON must not contain %q, got %s", key, string(data))
		}
	}
	if m["registration_id"] != float64(1) || m["approved"] != true {
		t.Fatalf("unexpected payload: %s", string(data))
	}
}
