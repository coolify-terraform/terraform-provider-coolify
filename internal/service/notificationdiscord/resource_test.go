package notificationdiscord_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockDiscord struct {
	mu          sync.Mutex
	Enabled     bool
	Webhook     string
	PingEnabled bool
	HideWebhook bool
	notificationcommon.EventStore
}

func (m *mockDiscord) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":                   1,
		"team_id":              0,
		"discord_enabled":      m.Enabled,
		"discord_ping_enabled": m.PingEnabled,
	}
	m.PutSnapshot(out, "discord")
	if !m.HideWebhook {
		out["discord_webhook_url"] = m.Webhook
	}
	return out
}

func newMockServer(store *mockDiscord) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/discord", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		allowed := notificationcommon.MergeAllowed(
			notificationcommon.EventAllowedFields("discord"),
			"discord_enabled", "discord_webhook_url", "discord_ping_enabled",
		)
		if notificationcommon.RejectUnknownFields(w, body, allowed) {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "discord_enabled", &store.Enabled)
		notificationcommon.StringFromBody(body, "discord_webhook_url", &store.Webhook)
		notificationcommon.BoolFromBody(body, "discord_ping_enabled", &store.PingEnabled)
		store.ApplyBody("discord", body)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestDiscordNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockDiscord{EventStore: notificationcommon.EventStore{DeploymentFailure: true, BackupFailure: true, ServerDiskUsage: true}}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/123/abc"
  deployment_failure = true
  backup_failure     = false
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "webhook_url", "https://discord.com/api/webhooks/123/abc"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "deployment_failure", "true"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "backup_failure", "false"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if !store.Enabled {
							t.Fatal("expected discord_enabled true in PATCH body effect")
						}
						if store.Webhook != "https://discord.com/api/webhooks/123/abc" {
							t.Fatalf("webhook = %q", store.Webhook)
						}
						if store.BackupFailure {
							t.Fatal("expected backup_failure false")
						}
						return nil
					},
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/123/abc"
  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = false
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "backup_failure", "true"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "server_disk_usage", "false"),
				),
			},
			{
				ResourceName:            "coolify_notification_discord.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
		},
	})
}

func TestDiscordNotificationResource_PreserveHiddenWebhook(t *testing.T) {
	t.Parallel()
	store := &mockDiscord{HideWebhook: true}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/999/secret"
`),
				Check: resource.TestCheckResourceAttr(
					"coolify_notification_discord.test",
					"webhook_url",
					"https://discord.com/api/webhooks/999/secret",
				),
			},
		},
	})
}

func TestDiscordNotificationResource_InvalidImport(t *testing.T) {
	t.Parallel()
	store := &mockDiscord{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/1/x"
`),
			},
			{
				ResourceName:  "coolify_notification_discord.test",
				ImportState:   true,
				ImportStateId: "not-current",
				ExpectError:   regexp.MustCompile(`team singleton|import with id "current"`),
			},
		},
	})
}

func TestDiscordNotificationResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	store := &mockDiscord{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.Enabled {
				return fmt.Errorf("expected discord_enabled false after destroy")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/123/abc"
  deployment_failure = true
`),
				Check: resource.TestCheckResourceAttr("coolify_notification_discord.test", "enabled", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}

func TestDiscordNotificationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"discord_enabled":false}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/1/x"
`),
				ExpectError: regexp.MustCompile(`Error configuring Discord notifications`),
			},
		},
	})
}

func TestDiscordNotificationResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		// Post-create refresh is get #1 after patch; fail subsequent Reads (step 2).
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"discord_enabled":true,"discord_webhook_url":"https://discord.com/api/webhooks/1/x"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"discord_enabled":true,"discord_webhook_url":"https://discord.com/api/webhooks/1/x"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/1/x"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/1/x"
`),
				ExpectError: regexp.MustCompile(`Error reading Discord notifications`),
			},
		},
	})
}

func TestDiscordNotificationResource_UpdateAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"discord_enabled":true,"discord_webhook_url":"https://discord.com/api/webhooks/1/x"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"discord_enabled":true,"discord_webhook_url":"https://discord.com/api/webhooks/1/x"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/1/x"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_discord", "test", `
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/1/x"
  deployment_failure = true
`),
				ExpectError: regexp.MustCompile(`Error updating Discord notifications`),
			},
		},
	})
}
