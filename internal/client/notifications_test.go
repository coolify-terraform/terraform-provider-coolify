package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_DiscordNotifications_GetUpdate(t *testing.T) {
	t.Parallel()
	var lastPatch map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"discord_enabled":                          true,
			"discord_webhook_url":                      "https://discord.com/api/webhooks/1/x",
			"discord_ping_enabled":                     false,
			"deployment_failure_discord_notifications": true,
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/discord", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"discord_enabled":                          lastPatch["discord_enabled"],
			"discord_webhook_url":                      "https://discord.com/api/webhooks/1/x",
			"deployment_failure_discord_notifications": true,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	ctx := context.Background()

	got, err := c.GetDiscordNotifications(ctx)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, "https://discord.com/api/webhooks/1/x", got.Webhook)

	en := false
	_, err = c.UpdateDiscordNotifications(ctx, client.UpdateDiscordNotificationInput{Enabled: &en})
	require.NoError(t, err)
	assert.Equal(t, false, lastPatch["discord_enabled"])
}

func TestClient_SlackNotifications_GetUpdate(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"slack_enabled":     false,
			"slack_webhook_url": "https://example.com/coolify-slack-webhook",
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"slack_enabled":     body["slack_enabled"],
			"slack_webhook_url": body["slack_webhook_url"],
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	ctx := context.Background()
	got, err := c.GetSlackNotifications(ctx)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
	en := true
	wh := "https://example.com/coolify-slack-webhook-updated"
	out, err := c.UpdateSlackNotifications(ctx, client.UpdateSlackNotificationInput{Enabled: &en, Webhook: &wh})
	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, wh, out.Webhook)
}
