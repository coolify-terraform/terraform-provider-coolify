package notificationwebhook_test

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

type mockWebhook struct {
	mu      sync.Mutex
	Enabled bool
	Webhook string
	notificationcommon.EventStore
}

func (m *mockWebhook) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":              1,
		"team_id":         0,
		"webhook_enabled": m.Enabled,
		"webhook_url":     m.Webhook,
	}
	m.PutSnapshot(out, "webhook")
	return out
}

func newMockServer(store *mockWebhook) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "webhook_enabled", &store.Enabled)
		notificationcommon.StringFromBody(body, "webhook_url", &store.Webhook)
		store.ApplyBody("webhook", body)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestWebhookNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockWebhook{EventStore: notificationcommon.EventStore{DeploymentFailure: true, BackupFailure: true, ServerDiskUsage: true}}
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

func TestWebhookNotificationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"webhook_enabled":false}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled     = true
  webhook_url = "https://example.com/hook"
`),
				ExpectError: regexp.MustCompile(`Error configuring Webhook notifications`),
			},
		},
	})
}

func TestWebhookNotificationResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		// Post-create refresh is get #1 after patch; fail subsequent Reads (step 2).
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"webhook_enabled":true,"webhook_url":"https://example.com/hook"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"webhook_enabled":true,"webhook_url":"https://example.com/hook"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
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
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled     = true
  webhook_url = "https://example.com/hook"
`),
				ExpectError: regexp.MustCompile(`Error reading Webhook notifications`),
			},
		},
	})
}

func TestWebhookNotificationResource_UpdateAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"webhook_enabled":true,"webhook_url":"https://example.com/hook"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/webhook", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"webhook_enabled":true,"webhook_url":"https://example.com/hook"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
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
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_webhook", "test", `
  enabled     = true
  webhook_url = "https://example.com/hook"
  deployment_failure = true
`),
				ExpectError: regexp.MustCompile(`Error updating Webhook notifications`),
			},
		},
	})
}
