package notificationpushover_test

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

type mockPushover struct {
	mu      sync.Mutex
	Enabled bool
	UserKey string
	Token   string
	notificationcommon.EventStore
}

func (m *mockPushover) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":                 1,
		"team_id":            0,
		"pushover_enabled":   m.Enabled,
		"pushover_user_key":  m.UserKey,
		"pushover_api_token": m.Token,
	}
	m.PutSnapshot(out, "pushover")
	return out
}

func newMockServer(store *mockPushover) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "pushover_enabled", &store.Enabled)
		notificationcommon.StringFromBody(body, "pushover_user_key", &store.UserKey)
		notificationcommon.StringFromBody(body, "pushover_api_token", &store.Token)
		store.ApplyBody("pushover", body)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestPushoverNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockPushover{EventStore: notificationcommon.EventStore{DeploymentFailure: true}}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled            = true
  user_key           = "u-test-user-key"
  api_token          = "a-test-api-token"
  deployment_failure = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "user_key", "u-test-user-key"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if store.UserKey != "u-test-user-key" || store.Token != "a-test-api-token" {
							t.Fatalf("keys=%q token=%q", store.UserKey, store.Token)
						}
						return nil
					},
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled            = false
  user_key           = "u-test-user-key"
  api_token          = "a-test-api-token"
  deployment_failure = true
  backup_failure     = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "backup_failure", "true"),
				),
			},
			{
				ResourceName:            "coolify_notification_pushover.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"user_key", "api_token"},
			},
		},
	})
}

func TestPushoverNotificationResource_InvalidImport(t *testing.T) {
	t.Parallel()
	store := &mockPushover{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u"
  api_token = "t"
`),
			},
			{
				ResourceName:  "coolify_notification_pushover.test",
				ImportState:   true,
				ImportStateId: "not-current",
				ExpectError:   regexp.MustCompile(`team singleton|import with id "current"`),
			},
		},
	})
}

func TestPushoverNotificationResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	store := &mockPushover{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.Enabled {
				return fmt.Errorf("expected pushover_enabled false after destroy")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u-test-user-key"
  api_token = "a-test-api-token"
  deployment_failure = true
`),
				Check: resource.TestCheckResourceAttr("coolify_notification_pushover.test", "enabled", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}

func TestPushoverNotificationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":false}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u"
  api_token = "t"
`),
				ExpectError: regexp.MustCompile(`Error configuring Pushover notifications`),
			},
		},
	})
}

func TestPushoverNotificationResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		// Post-create refresh is get #1 after patch; fail subsequent Reads (step 2).
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u","pushover_api_token":"t"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u","pushover_api_token":"t"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u"
  api_token = "t"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u"
  api_token = "t"
`),
				ExpectError: regexp.MustCompile(`Error reading Pushover notifications`),
			},
		},
	})
}

func TestPushoverNotificationResource_UpdateAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled  = true
  user_key = "u1"
  api_token = "t1"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled  = true
  user_key = "u1"
  api_token = "t1"
  deployment_failure = true
`),
				ExpectError: regexp.MustCompile(`Error updating Pushover notifications`),
			},
		},
	})
}

func TestPushoverNotificationResource_DestroyAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		// Create succeeds (patch 1); destroy disable fails (patch 2+).
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u1"
  api_token = "t1"
`),
			},
			{
				Config:      acctest.ProviderBlockForURL(srv.URL),
				ExpectError: regexp.MustCompile(`Error disabling Pushover notifications on destroy`),
			},
		},
	})
}

func TestPushoverNotificationResource_ReadNotFound(t *testing.T) {
	t.Parallel()
	var gone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		if gone {
			http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u1"
  api_token = "t1"
`),
			},
			{
				PreConfig: func() { gone = true },
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u1"
  api_token = "t1"
`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPushoverNotificationResource_DestroyNotFound(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches >= 2 {
			http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"pushover_enabled":true,"pushover_user_key":"u1","pushover_api_token":"t1"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_pushover", "test", `
  enabled   = true
  user_key  = "u1"
  api_token = "t1"
`),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}
