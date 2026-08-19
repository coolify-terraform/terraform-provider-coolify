package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
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
	var lastPatch map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"slack_enabled":     false,
			"slack_webhook_url": "https://example.com/coolify-slack-webhook",
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"slack_enabled":     lastPatch["slack_enabled"],
			"slack_webhook_url": lastPatch["slack_webhook_url"],
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
	assert.Equal(t, true, lastPatch["slack_enabled"])
	assert.Equal(t, wh, lastPatch["slack_webhook_url"])
}

func TestClient_EmailNotifications_GetUpdate(t *testing.T) {
	t.Parallel()
	var lastPatch map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"smtp_enabled": true,
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"smtp_enabled": lastPatch["smtp_enabled"],
			"smtp_port":    lastPatch["smtp_port"],
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	ctx := context.Background()
	got, err := c.GetEmailNotifications(ctx)
	require.NoError(t, err)
	assert.True(t, got.SMTPEnabled)
	require.NotNil(t, got.SMTPPort)
	assert.Equal(t, 587, *got.SMTPPort)
	en := false
	port := 465
	_, err = c.UpdateEmailNotifications(ctx, client.UpdateEmailNotificationInput{SMTPEnabled: &en, SMTPPort: &port})
	require.NoError(t, err)
	assert.Equal(t, false, lastPatch["smtp_enabled"])
	assert.Equal(t, float64(465), lastPatch["smtp_port"])
}

func TestClient_GetEmailNotifications_NullEhloDomain(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":false,"smtp_ehlo_domain":null}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	got, err := c.GetEmailNotifications(context.Background())
	require.NoError(t, err)
	assert.Nil(t, got.SMTPEhloDomain)
}

func TestClient_EmailNotifications_SMTPEhloDomainVersionGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		wantKey bool
	}{
		{"4.3.9 withholds", "4.3.9", false},
		{"4.3.10 sends", "4.3.10", true},
		{"4.4-rc.1 sends", "4.4-rc.1", true},
		{"empty version sends", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var lastPatch map[string]interface{}
			mux := http.NewServeMux()
			mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
				require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id": 1, "team_id": 0, "smtp_enabled": true,
					"smtp_ehlo_domain": lastPatch["smtp_ehlo_domain"],
				})
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()
			c := client.New(srv.URL, "tok")
			c.CoolifyVersion = tt.version
			ehlo := "mail.example.com"
			_, err := c.UpdateEmailNotifications(context.Background(), client.UpdateEmailNotificationInput{
				SMTPEhloDomain: &ehlo,
			})
			require.NoError(t, err)
			_, sent := lastPatch["smtp_ehlo_domain"]
			if sent != tt.wantKey {
				t.Errorf("Coolify %q: smtp_ehlo_domain sent=%v, want %v (body=%v)",
					tt.version, sent, tt.wantKey, lastPatch)
			}
		})
	}
}

func TestClient_TelegramNotifications_GetUpdate(t *testing.T) {
	t.Parallel()
	var lastPatch map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"telegram_enabled": false,
			"telegram_token":   "tok",
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/telegram", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"telegram_enabled": lastPatch["telegram_enabled"],
			"telegram_token":   lastPatch["telegram_token"],
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	ctx := context.Background()
	got, err := c.GetTelegramNotifications(ctx)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
	en := true
	tok := "new"
	out, err := c.UpdateTelegramNotifications(ctx, client.UpdateTelegramNotificationInput{Enabled: &en, Token: &tok})
	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, "new", out.Token)
	assert.Equal(t, true, lastPatch["telegram_enabled"])
	assert.Equal(t, "new", lastPatch["telegram_token"])
}

func TestClient_WebhookNotifications_GetUpdate(t *testing.T) {
	t.Parallel()
	var lastPatch map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"webhook_enabled": true,
			"webhook_url":     "https://example.com/hooks/coolify",
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/webhook", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"webhook_enabled": lastPatch["webhook_enabled"],
			"webhook_url":     lastPatch["webhook_url"],
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	ctx := context.Background()

	got, err := c.GetWebhookNotifications(ctx)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, "https://example.com/hooks/coolify", got.Webhook)

	en := false
	wh := "https://example.com/hooks/coolify-updated"
	out, err := c.UpdateWebhookNotifications(ctx, client.UpdateWebhookNotificationInput{Enabled: &en, Webhook: &wh})
	require.NoError(t, err)
	assert.False(t, out.Enabled)
	assert.Equal(t, wh, out.Webhook)
	assert.Equal(t, false, lastPatch["webhook_enabled"])
	assert.Equal(t, wh, lastPatch["webhook_url"])
}

func TestClient_PushoverNotifications_GetUpdate(t *testing.T) {
	t.Parallel()
	var lastPatch map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"pushover_enabled":   false,
			"pushover_user_key":  "user-key",
			"pushover_api_token": "api-token",
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastPatch))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"pushover_enabled":   lastPatch["pushover_enabled"],
			"pushover_user_key":  lastPatch["pushover_user_key"],
			"pushover_api_token": lastPatch["pushover_api_token"],
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "tok")
	ctx := context.Background()

	got, err := c.GetPushoverNotifications(ctx)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
	assert.Equal(t, "user-key", got.UserKey)

	en := true
	user := "new-user"
	tok := "new-token"
	out, err := c.UpdatePushoverNotifications(ctx, client.UpdatePushoverNotificationInput{
		Enabled: &en, UserKey: &user, APIToken: &tok,
	})
	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, "new-user", out.UserKey)
	assert.Equal(t, "new-token", out.APIToken)
	assert.Equal(t, true, lastPatch["pushover_enabled"])
	assert.Equal(t, "new-user", lastPatch["pushover_user_key"])
	assert.Equal(t, "new-token", lastPatch["pushover_api_token"])
}

func TestNotificationUpdateJSONTags(t *testing.T) {
	t.Parallel()
	for _, ch := range []string{"email", "discord", "slack", "telegram", "pushover", "webhook"} {
		ch := ch
		t.Run(ch, func(t *testing.T) {
			t.Parallel()
			tags := client.NotificationUpdateJSONTags(ch)
			require.NotEmpty(t, tags, "channel %s should expose update JSON tags", ch)
			// Every channel uses an enabled-style field (discord_enabled, smtp_enabled, …).
			hasEnabled := false
			for k := range tags {
				if len(k) >= 7 && (k[len(k)-7:] == "enabled" || k == "smtp_enabled" || k == "resend_enabled" || k == "use_instance_email_settings") {
					hasEnabled = true
					break
				}
			}
			assert.True(t, hasEnabled, "tags for %s: %v", ch, tags)
		})
	}
	assert.Nil(t, client.NotificationUpdateJSONTags("unknown-channel"))
	assert.Nil(t, client.NotificationUpdateJSONTags(""))
}

func TestNotificationThreadJSONTags_CoverTelegramThreads(t *testing.T) {
	t.Parallel()
	tags := client.NotificationUpdateJSONTags("telegram")
	require.NotEmpty(t, tags)
	for _, ev := range notificationcommon.EventAttributeNames() {
		key := notificationcommon.ThreadJSONKey("telegram", ev)
		_, ok := tags[key]
		assert.True(t, ok, "telegram update input missing thread JSON key %q", key)
	}
}

func TestNotificationEventJSONTags_CoverSharedEvents(t *testing.T) {
	t.Parallel()
	// Short schema names from notificationcommon must appear in each channel's
	// Update* JSON tags (as <event>_<channel>_notifications or similar).
	events := notificationcommon.EventAttributeNames()
	for _, ch := range []string{"discord", "slack", "telegram", "pushover", "webhook", "email"} {
		ch := ch
		t.Run(ch, func(t *testing.T) {
			t.Parallel()
			tags := client.NotificationUpdateJSONTags(ch)
			require.NotEmpty(t, tags)
			joined := ""
			for k := range tags {
				joined += k + " "
			}
			for _, ev := range events {
				if !strings.Contains(joined, ev) {
					t.Errorf("channel %s missing event %q in update JSON tags: %v", ch, ev, tags)
				}
			}
		})
	}
}
