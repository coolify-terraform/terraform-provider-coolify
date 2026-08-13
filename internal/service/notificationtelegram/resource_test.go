package notificationtelegram_test

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

type mockTelegram struct {
	mu                sync.Mutex
	Enabled           bool
	Token             string
	ChatID            string
	DeploymentFailure bool
	BackupFailure     bool
	ThreadDeployFail  string
	HideSecrets       bool
}

func (m *mockTelegram) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id": 1, "team_id": 0,
		"telegram_enabled":                          m.Enabled,
		"deployment_failure_telegram_notifications": m.DeploymentFailure,
		"backup_failure_telegram_notifications":     m.BackupFailure,
	}
	if !m.HideSecrets {
		out["telegram_token"] = m.Token
		out["telegram_chat_id"] = m.ChatID
		out["telegram_notifications_deployment_failure_thread_id"] = m.ThreadDeployFail
	}
	return out
}

func newMockServer(store *mockTelegram) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/telegram", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"message":"bad json"}`, http.StatusBadRequest)
			return
		}
		allowed := map[string]bool{
			"telegram_enabled": true, "telegram_token": true, "telegram_chat_id": true,
			"deployment_success_telegram_notifications": true, "deployment_failure_telegram_notifications": true,
			"status_change_telegram_notifications": true, "backup_success_telegram_notifications": true,
			"backup_failure_telegram_notifications": true, "scheduled_task_success_telegram_notifications": true,
			"scheduled_task_failure_telegram_notifications": true, "docker_cleanup_success_telegram_notifications": true,
			"docker_cleanup_failure_telegram_notifications": true, "server_disk_usage_telegram_notifications": true,
			"server_reachable_telegram_notifications": true, "server_unreachable_telegram_notifications": true,
			"server_patch_telegram_notifications": true, "traefik_outdated_telegram_notifications": true,
			"telegram_notifications_deployment_success_thread_id": true, "telegram_notifications_deployment_failure_thread_id": true,
			"telegram_notifications_status_change_thread_id": true, "telegram_notifications_backup_success_thread_id": true,
			"telegram_notifications_backup_failure_thread_id": true, "telegram_notifications_scheduled_task_success_thread_id": true,
			"telegram_notifications_scheduled_task_failure_thread_id": true, "telegram_notifications_docker_cleanup_success_thread_id": true,
			"telegram_notifications_docker_cleanup_failure_thread_id": true, "telegram_notifications_server_disk_usage_thread_id": true,
			"telegram_notifications_server_reachable_thread_id": true, "telegram_notifications_server_unreachable_thread_id": true,
			"telegram_notifications_server_patch_thread_id": true, "telegram_notifications_traefik_outdated_thread_id": true,
		}
		for k := range body {
			if !allowed[k] {
				http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
				return
			}
		}
		store.mu.Lock()
		if v, ok := body["telegram_enabled"].(bool); ok {
			store.Enabled = v
		}
		if v, ok := body["telegram_token"].(string); ok {
			store.Token = v
		}
		if v, ok := body["telegram_chat_id"].(string); ok {
			store.ChatID = v
		}
		if v, ok := body["deployment_failure_telegram_notifications"].(bool); ok {
			store.DeploymentFailure = v
		}
		if v, ok := body["backup_failure_telegram_notifications"].(bool); ok {
			store.BackupFailure = v
		}
		if v, ok := body["telegram_notifications_deployment_failure_thread_id"].(string); ok {
			store.ThreadDeployFail = v
		}
		store.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestTelegramNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockTelegram{DeploymentFailure: true}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled                         = true
  token                           = "123456:ABC-DEF"
  chat_id                         = "-100123"
  deployment_failure              = true
  thread_deployment_failure       = "42"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "token", "123456:ABC-DEF"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "chat_id", "-100123"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "thread_deployment_failure", "42"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if store.Token != "123456:ABC-DEF" || store.ThreadDeployFail != "42" {
							t.Fatalf("store token=%q thr=%q", store.Token, store.ThreadDeployFail)
						}
						return nil
					},
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled                         = false
  token                           = "123456:ABC-DEF"
  chat_id                         = "-100123"
  deployment_failure              = true
  backup_failure                  = true
  thread_deployment_failure       = "99"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "backup_failure", "true"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "thread_deployment_failure", "99"),
				),
			},
			{
				ResourceName:            "coolify_notification_telegram.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "chat_id", "thread_deployment_failure"},
			},
		},
	})
}

func TestTelegramNotificationResource_PreserveHiddenSecrets(t *testing.T) {
	t.Parallel()
	store := &mockTelegram{HideSecrets: true}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled = true
  token   = "secret-token"
  chat_id = "chat-1"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "token", "secret-token"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "chat_id", "chat-1"),
				),
			},
		},
	})
}
