package notificationdiscord_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockDiscord struct {
	mu                   sync.Mutex
	Enabled              bool   `json:"discord_enabled"`
	Webhook              string `json:"discord_webhook_url"`
	PingEnabled          bool   `json:"discord_ping_enabled"`
	DeploymentSuccess    bool   `json:"deployment_success_discord_notifications"`
	DeploymentFailure    bool   `json:"deployment_failure_discord_notifications"`
	StatusChange         bool   `json:"status_change_discord_notifications"`
	BackupSuccess        bool   `json:"backup_success_discord_notifications"`
	BackupFailure        bool   `json:"backup_failure_discord_notifications"`
	ScheduledTaskSuccess bool   `json:"scheduled_task_success_discord_notifications"`
	ScheduledTaskFailure bool   `json:"scheduled_task_failure_discord_notifications"`
	DockerCleanupSuccess bool   `json:"docker_cleanup_success_discord_notifications"`
	DockerCleanupFailure bool   `json:"docker_cleanup_failure_discord_notifications"`
	ServerDiskUsage      bool   `json:"server_disk_usage_discord_notifications"`
	ServerReachable      bool   `json:"server_reachable_discord_notifications"`
	ServerUnreachable    bool   `json:"server_unreachable_discord_notifications"`
	ServerPatch          bool   `json:"server_patch_discord_notifications"`
	TraefikOutdated      bool   `json:"traefik_outdated_discord_notifications"`
	HideWebhook          bool   `json:"-"`
}

func (m *mockDiscord) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":                   1,
		"team_id":              0,
		"discord_enabled":      m.Enabled,
		"discord_ping_enabled": m.PingEnabled,
		"deployment_success_discord_notifications":     m.DeploymentSuccess,
		"deployment_failure_discord_notifications":     m.DeploymentFailure,
		"status_change_discord_notifications":          m.StatusChange,
		"backup_success_discord_notifications":         m.BackupSuccess,
		"backup_failure_discord_notifications":         m.BackupFailure,
		"scheduled_task_success_discord_notifications": m.ScheduledTaskSuccess,
		"scheduled_task_failure_discord_notifications": m.ScheduledTaskFailure,
		"docker_cleanup_success_discord_notifications": m.DockerCleanupSuccess,
		"docker_cleanup_failure_discord_notifications": m.DockerCleanupFailure,
		"server_disk_usage_discord_notifications":      m.ServerDiskUsage,
		"server_reachable_discord_notifications":       m.ServerReachable,
		"server_unreachable_discord_notifications":     m.ServerUnreachable,
		"server_patch_discord_notifications":           m.ServerPatch,
		"traefik_outdated_discord_notifications":       m.TraefikOutdated,
	}
	if !m.HideWebhook {
		out["discord_webhook_url"] = m.Webhook
	}
	return out
}

func newMockServer(store *mockDiscord) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/discord", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/discord", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"message":"bad json"}`, http.StatusBadRequest)
			return
		}
		// Reject unknown fields like Coolify does.
		allowed := map[string]bool{
			"discord_enabled": true, "discord_webhook_url": true, "discord_ping_enabled": true,
			"deployment_success_discord_notifications": true, "deployment_failure_discord_notifications": true,
			"status_change_discord_notifications": true, "backup_success_discord_notifications": true,
			"backup_failure_discord_notifications": true, "scheduled_task_success_discord_notifications": true,
			"scheduled_task_failure_discord_notifications": true, "docker_cleanup_success_discord_notifications": true,
			"docker_cleanup_failure_discord_notifications": true, "server_disk_usage_discord_notifications": true,
			"server_reachable_discord_notifications": true, "server_unreachable_discord_notifications": true,
			"server_patch_discord_notifications": true, "traefik_outdated_discord_notifications": true,
		}
		for k := range body {
			if !allowed[k] {
				http.Error(w, `{"message":"Validation failed.","errors":{"`+k+`":["This field is not allowed."]}}`, http.StatusUnprocessableEntity)
				return
			}
		}
		store.mu.Lock()
		if v, ok := body["discord_enabled"].(bool); ok {
			store.Enabled = v
		}
		if v, ok := body["discord_webhook_url"].(string); ok {
			store.Webhook = v
		}
		if v, ok := body["discord_ping_enabled"].(bool); ok {
			store.PingEnabled = v
		}
		setB := func(key string, dst *bool) {
			if v, ok := body[key].(bool); ok {
				*dst = v
			}
		}
		setB("deployment_success_discord_notifications", &store.DeploymentSuccess)
		setB("deployment_failure_discord_notifications", &store.DeploymentFailure)
		setB("status_change_discord_notifications", &store.StatusChange)
		setB("backup_success_discord_notifications", &store.BackupSuccess)
		setB("backup_failure_discord_notifications", &store.BackupFailure)
		setB("scheduled_task_success_discord_notifications", &store.ScheduledTaskSuccess)
		setB("scheduled_task_failure_discord_notifications", &store.ScheduledTaskFailure)
		setB("docker_cleanup_success_discord_notifications", &store.DockerCleanupSuccess)
		setB("docker_cleanup_failure_discord_notifications", &store.DockerCleanupFailure)
		setB("server_disk_usage_discord_notifications", &store.ServerDiskUsage)
		setB("server_reachable_discord_notifications", &store.ServerReachable)
		setB("server_unreachable_discord_notifications", &store.ServerUnreachable)
		setB("server_patch_discord_notifications", &store.ServerPatch)
		setB("traefik_outdated_discord_notifications", &store.TraefikOutdated)
		store.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestDiscordNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockDiscord{
		DeploymentFailure: true,
		BackupFailure:     true,
		ServerDiskUsage:   true,
	}
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
