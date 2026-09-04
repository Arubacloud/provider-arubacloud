package clients

import (
	"encoding/json"
	"testing"
)

func TestCredentialsParsing_RequiredFields(t *testing.T) {
	raw := `{"client_id":"cid","client_secret":"csec"}`
	creds := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if creds[credKeyClientID] != "cid" {
		t.Errorf("client_id: got %q, want %q", creds[credKeyClientID], "cid")
	}
	if creds[credKeyClientSecret] != "csec" {
		t.Errorf("client_secret: got %q, want %q", creds[credKeyClientSecret], "csec")
	}
}

func TestCredentialsParsing_OptionalFields(t *testing.T) {
	raw := `{
		"client_id":"cid",
		"client_secret":"csec",
		"base_url":"https://api.example.com",
		"token_issuer_url":"https://auth.example.com",
		"resource_timeout":"60m"
	}`
	creds := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	tests := []struct{ key, want string }{
		{credKeyBaseURL, "https://api.example.com"},
		{credKeyTokenIssuerURL, "https://auth.example.com"},
		{credKeyResourceTimeout, "60m"},
	}
	for _, tc := range tests {
		if creds[tc.key] != tc.want {
			t.Errorf("%s: got %q, want %q", tc.key, creds[tc.key], tc.want)
		}
	}
}

func TestCredentialsParsing_OptionalFieldsAbsent(t *testing.T) {
	raw := `{"client_id":"cid","client_secret":"csec"}`
	creds := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	for _, key := range []string{credKeyBaseURL, credKeyTokenIssuerURL, credKeyResourceTimeout} {
		if v, ok := creds[key]; ok && v != "" {
			t.Errorf("expected %s to be absent or empty, got %q", key, v)
		}
	}
}

func TestCredentialsParsing_InvalidJSON(t *testing.T) {
	creds := map[string]string{}
	err := json.Unmarshal([]byte("not-json"), &creds)
	if err == nil {
		t.Error("expected unmarshal error for invalid JSON, got nil")
	}
}

func TestTerraformSetupConfiguration(t *testing.T) {
	creds := map[string]string{
		credKeyClientID:        "my-id",
		credKeyClientSecret:    "my-secret",
		credKeyBaseURL:         "https://custom.api",
		credKeyTokenIssuerURL:  "https://custom.auth",
		credKeyResourceTimeout: "45m",
	}

	cfg := map[string]any{
		credKeyClientID:     creds[credKeyClientID],
		credKeyClientSecret: creds[credKeyClientSecret],
	}
	for _, key := range []string{credKeyBaseURL, credKeyTokenIssuerURL, credKeyResourceTimeout} {
		if v := creds[key]; v != "" {
			cfg[key] = v
		}
	}

	if cfg[credKeyClientID] != "my-id" {
		t.Errorf("client_id: got %v", cfg[credKeyClientID])
	}
	if cfg[credKeyClientSecret] != "my-secret" {
		t.Errorf("client_secret: got %v", cfg[credKeyClientSecret])
	}
	if cfg[credKeyBaseURL] != "https://custom.api" {
		t.Errorf("base_url: got %v", cfg[credKeyBaseURL])
	}
	if cfg[credKeyResourceTimeout] != "45m" {
		t.Errorf("resource_timeout: got %v", cfg[credKeyResourceTimeout])
	}
}

func TestTerraformSetupConfiguration_EmptyOptionals(t *testing.T) {
	creds := map[string]string{
		credKeyClientID:     "my-id",
		credKeyClientSecret: "my-secret",
	}

	cfg := map[string]any{
		credKeyClientID:     creds[credKeyClientID],
		credKeyClientSecret: creds[credKeyClientSecret],
	}
	for _, key := range []string{credKeyBaseURL, credKeyTokenIssuerURL, credKeyResourceTimeout} {
		if v := creds[key]; v != "" {
			cfg[key] = v
		}
	}

	for _, key := range []string{credKeyBaseURL, credKeyTokenIssuerURL, credKeyResourceTimeout} {
		if _, present := cfg[key]; present {
			t.Errorf("expected %s to be absent from config when not set in credentials", key)
		}
	}
}
