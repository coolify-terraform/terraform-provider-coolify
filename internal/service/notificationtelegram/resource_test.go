package notificationtelegram_test

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

type mockTelegram struct {
	mu               sync.Mutex
	Enabled          bool
	Token            string
	ChatID           string
	ThreadDeployFail string
	HideSecrets      bool
	notificationcommon.EventStore
}

func (m *mockTelegram) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":               1,
		"team_id":          0,
		"telegram_enabled": m.Enabled,
	}
	m.PutSnapshot(out, "telegram")
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
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/telegram", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		allowed := notificationcommon.MergeAllowed(
			notificationcommon.EventAllowedFields("telegram"),
			"telegram_enabled", "telegram_token", "telegram_chat_id",
			"telegram_notifications_deployment_success_thread_id",
			"telegram_notifications_deployment_failure_thread_id",
			"telegram_notifications_status_change_thread_id",
			"telegram_notifications_backup_success_thread_id",
			"telegram_notifications_backup_failure_thread_id",
			"telegram_notifications_scheduled_task_success_thread_id",
			"telegram_notifications_scheduled_task_failure_thread_id",
			"telegram_notifications_docker_cleanup_success_thread_id",
			"telegram_notifications_docker_cleanup_failure_thread_id",
			"telegram_notifications_server_disk_usage_thread_id",
			"telegram_notifications_server_reachable_thread_id",
			"telegram_notifications_server_unreachable_thread_id",
			"telegram_notifications_server_patch_thread_id",
			"telegram_notifications_traefik_outdated_thread_id",
		)
		if notificationcommon.RejectUnknownFields(w, body, allowed) {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "telegram_enabled", &store.Enabled)
		notificationcommon.StringFromBody(body, "telegram_token", &store.Token)
		notificationcommon.StringFromBody(body, "telegram_chat_id", &store.ChatID)
		notificationcommon.StringFromBody(body, "telegram_notifications_deployment_failure_thread_id", &store.ThreadDeployFail)
		store.ApplyBody("telegram", body)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestTelegramNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockTelegram{EventStore: notificationcommon.EventStore{DeploymentFailure: true}}
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

func TestTelegramNotificationResource_InvalidImport(t *testing.T) {
	t.Parallel()
	store := &mockTelegram{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled = true
  token   = "tok"
  chat_id = "1"
`),
			},
			{
				ResourceName:  "coolify_notification_telegram.test",
				ImportState:   true,
				ImportStateId: "not-current",
				ExpectError:   regexp.MustCompile(`team singleton|import with id "current"`),
			},
		},
	})
}

func TestTelegramNotificationResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	store := &mockTelegram{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.Enabled {
				return fmt.Errorf("expected telegram_enabled false after destroy")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled = true
  token   = "123456:ABC-DEF"
  chat_id = "-100123"
  deployment_failure = true
`),
				Check: resource.TestCheckResourceAttr("coolify_notification_telegram.test", "enabled", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}

func TestTelegramNotificationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"telegram_enabled":false}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled = true
  token   = "tok"
  chat_id = "1"
`),
				ExpectError: regexp.MustCompile(`Error configuring Telegram notifications`),
			},
		},
	})
}

func TestTelegramNotificationResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		// Post-create refresh is get #1 after patch; fail subsequent Reads (step 2).
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"telegram_enabled":true,"telegram_token":"tok","telegram_chat_id":"1"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"telegram_enabled":true,"telegram_token":"tok","telegram_chat_id":"1"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled = true
  token   = "tok"
  chat_id = "1"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_telegram", "test", `
  enabled = true
  token   = "tok"
  chat_id = "1"
`),
				ExpectError: regexp.MustCompile(`Error reading Telegram notifications`),
			},
		},
	})
}
