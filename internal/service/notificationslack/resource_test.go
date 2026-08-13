package notificationslack_test

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

type mockSlack struct {
	mu      sync.Mutex
	Enabled bool
	Webhook string
	notificationcommon.EventStore
}

func (m *mockSlack) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":                1,
		"team_id":           0,
		"slack_enabled":     m.Enabled,
		"slack_webhook_url": m.Webhook,
	}
	m.PutSnapshot(out, "slack")
	return out
}

func newMockServer(store *mockSlack) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "slack_enabled", &store.Enabled)
		notificationcommon.StringFromBody(body, "slack_webhook_url", &store.Webhook)
		store.ApplyBody("slack", body)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestSlackNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockSlack{EventStore: notificationcommon.EventStore{DeploymentFailure: true, BackupFailure: true, ServerDiskUsage: true}}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled            = true
  webhook_url        = "https://example.com/coolify-slack-webhook"
  deployment_failure = true
  status_change      = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "webhook_url", "https://example.com/coolify-slack-webhook"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "status_change", "true"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if store.Webhook != "https://example.com/coolify-slack-webhook" {
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
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled            = false
  webhook_url        = "https://example.com/coolify-slack-webhook"
  deployment_failure = true
  status_change      = false
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "status_change", "false"),
				),
			},
			{
				ResourceName:            "coolify_notification_slack.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
		},
	})
}

func TestSlackNotificationResource_InvalidImport(t *testing.T) {
	t.Parallel()
	store := &mockSlack{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://example.com/slack"
`),
			},
			{
				ResourceName:  "coolify_notification_slack.test",
				ImportState:   true,
				ImportStateId: "not-current",
				ExpectError:   regexp.MustCompile(`team singleton|import with id "current"`),
			},
		},
	})
}

func TestSlackNotificationResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	store := &mockSlack{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.Enabled {
				return fmt.Errorf("expected slack_enabled false after destroy")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://example.com/coolify-slack-webhook"
  deployment_failure = true
`),
				Check: resource.TestCheckResourceAttr("coolify_notification_slack.test", "enabled", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}

func TestSlackNotificationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":false}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://example.com/slack"
`),
				ExpectError: regexp.MustCompile(`Error configuring Slack notifications`),
			},
		},
	})
}

func TestSlackNotificationResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		// Post-create refresh is get #1 after patch; fail subsequent Reads (step 2).
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://example.com/slack"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://example.com/slack"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://example.com/slack"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://example.com/slack"
`),
				ExpectError: regexp.MustCompile(`Error reading Slack notifications`),
			},
		},
	})
}

func TestSlackNotificationResource_UpdateAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://hooks.slack.com/services/T/B/x"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://hooks.slack.com/services/T/B/x"
  deployment_failure = true
`),
				ExpectError: regexp.MustCompile(`Error updating Slack notifications`),
			},
		},
	})
}

func TestSlackNotificationResource_DestroyAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		// Create succeeds (patch 1); destroy disable fails (patch 2+).
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://hooks.slack.com/services/T/B/x"
`),
			},
			{
				Config:      acctest.ProviderBlockForURL(srv.URL),
				ExpectError: regexp.MustCompile(`Error disabling Slack notifications on destroy`),
			},
		},
	})
}

func TestSlackNotificationResource_ReadNotFound(t *testing.T) {
	t.Parallel()
	var gone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		if gone {
			http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://hooks.slack.com/services/T/B/x"
`),
			},
			{
				PreConfig: func() { gone = true },
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://hooks.slack.com/services/T/B/x"
`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestSlackNotificationResource_DestroyNotFound(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/slack", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches >= 2 {
			http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"slack_enabled":true,"slack_webhook_url":"https://hooks.slack.com/services/T/B/x"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_slack", "test", `
  enabled     = true
  webhook_url = "https://hooks.slack.com/services/T/B/x"
`),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}
