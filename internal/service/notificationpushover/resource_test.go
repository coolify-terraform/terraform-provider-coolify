package notificationpushover_test

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

type mockPushover struct {
	mu                sync.Mutex
	Enabled           bool
	UserKey           string
	Token             string
	DeploymentFailure bool
	BackupFailure     bool
}

func (m *mockPushover) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]interface{}{
		"id": 1, "team_id": 0,
		"pushover_enabled":                          m.Enabled,
		"pushover_user_key":                         m.UserKey,
		"pushover_api_token":                        m.Token,
		"deployment_failure_pushover_notifications": m.DeploymentFailure,
		"backup_failure_pushover_notifications":     m.BackupFailure,
	}
}

func newMockServer(store *mockPushover) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/pushover", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/pushover", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		store.mu.Lock()
		if v, ok := body["pushover_enabled"].(bool); ok {
			store.Enabled = v
		}
		if v, ok := body["pushover_user_key"].(string); ok {
			store.UserKey = v
		}
		if v, ok := body["pushover_api_token"].(string); ok {
			store.Token = v
		}
		if v, ok := body["deployment_failure_pushover_notifications"].(bool); ok {
			store.DeploymentFailure = v
		}
		if v, ok := body["backup_failure_pushover_notifications"].(bool); ok {
			store.BackupFailure = v
		}
		store.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestPushoverNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockPushover{DeploymentFailure: true}
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
