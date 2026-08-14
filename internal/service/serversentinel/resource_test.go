package serversentinel_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestServerSentinelResource_CRUD(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"is_sentinel_enabled":       false,
		"is_metrics_enabled":        false,
		"is_sentinel_debug_enabled": false,
	}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/sentinel") {
			http.Error(w, r.URL.Path, http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode sentinel patch: %v", err)
			}
			if _, ok := body["is_sentinel_enabled"]; !ok {
				t.Errorf("expected is_sentinel_enabled in PATCH body, got %v", body)
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
resource "coolify_server_sentinel" "test" {
  server_uuid         = "` + serverUUID + `"
  is_sentinel_enabled = true
  is_metrics_enabled  = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_sentinel_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_metrics_enabled", "true"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_sentinel" "test" {
  server_uuid         = "` + serverUUID + `"
  is_sentinel_enabled = true
  is_metrics_enabled  = false
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_metrics_enabled", "false"),
			},
			{
				ResourceName:                         "coolify_server_sentinel.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}

// Destroy disables Sentinel remotely (no DELETE API).
func TestServerSentinelResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{"is_sentinel_enabled": false}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/sentinel") {
			http.Error(w, r.URL.Path, http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
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
		CheckDestroy: func(_ *terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if store["is_sentinel_enabled"] == true {
				return fmt.Errorf("expected Sentinel disabled after destroy, got %v", store["is_sentinel_enabled"])
			}
			return nil
		},
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_sentinel" "test" {
  server_uuid         = "` + serverUUID + `"
  is_sentinel_enabled = true
}`,
			Check: resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_sentinel_enabled", "true"),
		}},
	})
}

func TestServerSentinelResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sentinel") && r.Method == http.MethodPatch {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, r.URL.Path, http.StatusNotFound)
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_sentinel" "test" {
  server_uuid         = "aaaa0001-0001-4000-8000-000000000001"
  is_sentinel_enabled = true
}`,
			ExpectError: regexp.MustCompile(`Error applying Sentinel settings`),
		}},
	})
}
