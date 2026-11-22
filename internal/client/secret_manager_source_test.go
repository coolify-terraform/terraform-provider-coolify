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

func TestClient_SecretManagerSourceRoundTrip(t *testing.T) {
	t.Parallel()
	const (
		resourceType = "applications"
		uuid         = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
		token        = "11111111-1111-4111-8111-111111111111"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/{resource_type}/{uuid}/secret-manager", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, resourceType, r.PathValue("resource_type"))
		assert.Equal(t, uuid, r.PathValue("uuid"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{
			"integration_token_uuid": token,
			"settings":               map[string]any{},
		}, body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"integration_token_uuid": token,
			"provider":               "doppler",
			"settings":               nil,
		})
	})
	mux.HandleFunc("GET /api/v1/{resource_type}/{uuid}/secret-manager", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, resourceType, r.PathValue("resource_type"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"linked":                 false,
			"integration_token_uuid": nil,
			"provider":               nil,
			"settings":               nil,
			"available_tokens":       []map[string]string{{"uuid": token, "name": "doppler", "provider": "doppler"}},
		})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	err := c.SetSecretManagerSource(context.Background(), resourceType, uuid, client.SetSecretManagerSourceInput{
		IntegrationTokenUUID: token,
		Settings:             map[string]string{},
	})
	require.NoError(t, err)

	got, err := c.GetSecretManagerSource(context.Background(), resourceType, uuid)
	require.NoError(t, err)
	assert.False(t, got.Linked)
	assert.Nil(t, got.IntegrationTokenUUID)
	assert.Empty(t, got.Settings)
}

func TestClient_DeleteSecretManagerSourceNotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/{resource_type}/{uuid}/secret-manager", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"The resource has no secret manager source."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	err := c.DeleteSecretManagerSource(context.Background(), "services", "ffffffff-ffff-4fff-8fff-ffffffffffff")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
