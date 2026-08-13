package notificationwebhook_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockWebhook struct {
	mu                   sync.Mutex
	Enabled              bool
	Webhook              string
	DeploymentSuccess    bool
	DeploymentFailure    bool
	StatusChange         bool
	BackupSuccess        bool
	BackupFailure        bool
	ScheduledTaskSuccess bool
	ScheduledTaskFailure bool
	DockerCleanupSuccess bool
	DockerCleanupFailure bool
	ServerDiskUsage      bool
	ServerReachable      bool
	ServerUnreachable    bool
	ServerPatch          bool
	TraefikOutdated      bool
}

func (m *mockWebhook) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]interface{}{
		"id":              1,
		"team_id":         0,
		"webhook_enabled": m.Enabled,
		"webhook_url":     m.Webhook,
		"deployment_success_webhook_notifications":     m.DeploymentSuccess,
		"deployment_failure_webhook_notifications":     m.DeploymentFailure,
		"status_change_webhook_notifications":          m.StatusChange,
		"backup_success_webhook_notifications":         m.BackupSuccess,
		"backup_failure_webhook_notifications":         m.BackupFailure,
		"scheduled_task_success_webhook_notifications": m.ScheduledTaskSuccess,
		"scheduled_task_failure_webhook_notifications": m.ScheduledTaskFailure,
		"docker_cleanup_success_webhook_notifications": m.DockerCleanupSuccess,
		"docker_cleanup_failure_webhook_notifications": m.DockerCleanupFailure,
		"server_disk_usage_webhook_notifications":      m.ServerDiskUsage,
		"server_reachable_webhook_notifications":       m.ServerReachable,
		"server_unreachable_webhook_notifications":     m.ServerUnreachable,
		"server_patch_webhook_notifications":           m.ServerPatch,
		"traefik_outdated_webhook_notifications":       m.TraefikOutdated,
	}
}

func newMockServer(store *mockWebhook) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/webhook", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"message":"bad json"}`, http.StatusBadRequest)
			return
		}
		store.mu.Lock()
		if v, ok := body["webhook_enabled"].(bool); ok {
			store.Enabled = v
		}
		if v, ok := body["webhook_url"].(string); ok {
			store.Webhook = v
		}
		setB := func(key string, dst *bool) {
			if v, ok := body[key].(bool); ok {
				*dst = v
			}
		}
		setB("deployment_success_webhook_notifications", &store.DeploymentSuccess)
		setB("deployment_failure_webhook_notifications", &store.DeploymentFailure)
		setB("status_change_webhook_notifications", &store.StatusChange)
		setB("backup_success_webhook_notifications", &store.BackupSuccess)
		setB("backup_failure_webhook_notifications", &store.BackupFailure)
		setB("scheduled_task_success_webhook_notifications", &store.ScheduledTaskSuccess)
		setB("scheduled_task_failure_webhook_notifications", &store.ScheduledTaskFailure)
		setB("docker_cleanup_success_webhook_notifications", &store.DockerCleanupSuccess)
		setB("docker_cleanup_failure_webhook_notifications", &store.DockerCleanupFailure)
		setB("server_disk_usage_webhook_notifications", &store.ServerDiskUsage)
		setB("server_reachable_webhook_notifications", &store.ServerReachable)
		setB("server_unreachable_webhook_notifications", &store.ServerUnreachable)
		setB("server_patch_webhook_notifications", &store.ServerPatch)
		setB("traefik_outdated_webhook_notifications", &store.TraefikOutdated)
		store.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestWebhookNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockWebhook{DeploymentFailure: true, BackupFailure: true, ServerDiskUsage: true}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled            = true
  webhook_url        = "https://example.com/coolify-webhook"
  deployment_failure = true
  status_change      = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "webhook_url", "https://example.com/coolify-webhook"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "status_change", "true"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if store.Webhook != "https://example.com/coolify-webhook" {
							t.Fatalf("webhook = %q", store.Webhook)
						}
						if !store.StatusChange {
							t.Fatal("expected status_change true")
						}
						return nil
					},
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled            = false
  webhook_url        = "https://example.com/coolify-webhook"
  deployment_failure = true
  status_change      = false
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "status_change", "false"),
				),
			},
			{
				ResourceName:            "coolify_notification_webhook.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
		},
	})
}

func TestWebhookNotificationResource_InvalidImport(t *testing.T) {
	t.Parallel()
	store := &mockWebhook{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled     = true
  webhook_url = "https://example.com/hook"
`),
			},
			{
				ResourceName:  "coolify_notification_webhook.test",
				ImportState:   true,
				ImportStateId: "not-current",
				ExpectError:   regexp.MustCompile(`team singleton|import with id "current"`),
			},
		},
	})
}

func TestWebhookNotificationResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	store := &mockWebhook{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.Enabled {
				return fmt.Errorf("expected webhook_enabled false after destroy")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled     = true
  webhook_url = "https://example.com/coolify-webhook"
  deployment_failure = true
`),
				Check: resource.TestCheckResourceAttr("coolify_notification_webhook.test", "enabled", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}
