package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetInstanceEmailSettings(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"smtp_enabled":     true,
			"smtp_host":        "smtp.example.com",
			"smtp_port":        587,
			"smtp_encryption":  "starttls",
			"smtp_ehlo_domain": "mail.example.com",
			"resend_enabled":   false,
		})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	c.CoolifyVersion = "4.3.10"
	got, err := c.GetInstanceEmailSettings(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.SMTPEnabled)
	assert.Equal(t, "smtp.example.com", got.SMTPHost)
	require.NotNil(t, got.SMTPPort)
	assert.Equal(t, 587, *got.SMTPPort)
	require.NotNil(t, got.SMTPEhloDomain)
	assert.Equal(t, "mail.example.com", *got.SMTPEhloDomain)
}

func TestClient_UpdateInstanceEmailSettings(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, true, body["smtp_enabled"])
		assert.Equal(t, "smtp.example.com", body["smtp_host"])
		assert.EqualValues(t, 587, body["smtp_port"])
		_ = json.NewEncoder(w).Encode(body)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	c.CoolifyVersion = "4.3.10"
	en := true
	host := "smtp.example.com"
	port := 587
	got, err := c.UpdateInstanceEmailSettings(context.Background(), client.UpdateInstanceEmailInput{
		SMTPEnabled: &en,
		SMTPHost:    &host,
		SMTPPort:    &port,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.SMTPEnabled)
	assert.Equal(t, "smtp.example.com", got.SMTPHost)
}

func TestClient_GetInstanceEmailSettings_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not found."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	_, err := c.GetInstanceEmailSettings(context.Background())
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestClient_GetInstanceEmailSettings_APIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	_, err := c.GetInstanceEmailSettings(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting instance email settings")
	assert.False(t, client.IsNotFound(err))
}

func TestClient_UpdateInstanceEmailSettings_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not found."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	c.CoolifyVersion = "4.3.10"
	en := true
	_, err := c.UpdateInstanceEmailSettings(context.Background(), client.UpdateInstanceEmailInput{SMTPEnabled: &en})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
	assert.Contains(t, err.Error(), "updating instance email settings")
}

func TestClient_UpdateInstanceEmailSettings_APIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	en := true
	_, err := c.UpdateInstanceEmailSettings(context.Background(), client.UpdateInstanceEmailInput{SMTPEnabled: &en})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updating instance email settings")
}

func TestInstanceEmailUpdateJSONTags(t *testing.T) {
	t.Parallel()
	tags := client.InstanceEmailUpdateJSONTags()
	require.NotEmpty(t, tags)
	want := []string{
		"smtp_enabled", "smtp_from_address", "smtp_from_name", "smtp_host",
		"smtp_port", "smtp_encryption", "smtp_username", "smtp_password",
		"smtp_timeout", "smtp_ehlo_domain", "resend_enabled", "resend_api_key",
	}
	got := make([]string, 0, len(tags))
	for k := range tags {
		got = append(got, k)
	}
	assert.ElementsMatch(t, want, got)
}

func TestClient_SupportsInstanceEmailSettings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ver  string
		want bool
	}{
		{"", true},
		{"4.3.0", true},
		{"v4.3.0-edge", true},
		{"4.3.9", false},
		{"v4.3.9", false},
		{"4.3.10", true},
		{"v4.3.10", true},
		{"4.4-rc.1", true},
	}
	for _, tc := range cases {
		c := &client.Client{CoolifyVersion: tc.ver}
		assert.Equal(t, tc.want, c.SupportsInstanceEmailSettings(), tc.ver)
	}
}
