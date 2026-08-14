package serverlogdrain_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServerLogDrainResource_CRUD(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"is_logdrain_newrelic_enabled": false,
		"is_logdrain_axiom_enabled":    false,
		"is_logdrain_custom_enabled":   false,
	}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/log-drains") {
			http.Error(w, r.URL.Path, http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode log drain patch: %v", err)
			}
			if body["is_logdrain_axiom_enabled"] == nil && body["is_logdrain_newrelic_enabled"] == nil && body["is_logdrain_custom_enabled"] == nil {
				t.Errorf("expected a drain enable flag in PATCH body, got %v", body)
			}
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(store)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_log_drain" "test" {
  server_uuid                 = "` + serverUUID + `"
  is_logdrain_axiom_enabled   = true
  logdrain_axiom_dataset_name = "coolify"
  logdrain_axiom_api_key      = "axiom-key"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_log_drain.test", "is_logdrain_axiom_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_server_log_drain.test", "logdrain_axiom_dataset_name", "coolify"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_log_drain" "test" {
  server_uuid                 = "` + serverUUID + `"
  is_logdrain_axiom_enabled   = true
  logdrain_axiom_dataset_name = "coolify-prod"
  logdrain_axiom_api_key      = "axiom-key"
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_log_drain.test", "logdrain_axiom_dataset_name", "coolify-prod"),
			},
			{
				ResourceName:                         "coolify_server_log_drain.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
				ImportStateVerifyIgnore:              []string{"logdrain_axiom_api_key"},
			},
		},
	})
}
