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

func TestServerSentinelResource_CreateWhenEnableNotAllowed(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{"is_sentinel_enabled": true, "is_metrics_enabled": false}
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
			if _, ok := body["is_sentinel_enabled"]; ok {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"message":"Validation failed.","errors":{"is_sentinel_enabled":["This field is not allowed."]}}`))
				return
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
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_sentinel" "test" {
  server_uuid         = "` + serverUUID + `"
  is_sentinel_enabled = true
  is_metrics_enabled  = true
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_metrics_enabled", "true"),
				resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_sentinel_enabled", "true"),
			),
		}},
	})
}

func TestServerSentinelResource_IntervalDebugURLWrites(t *testing.T) {
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
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			if _, onlyEnable := body["is_sentinel_enabled"]; len(body) == 1 && onlyEnable {
				for k, v := range body {
					store[k] = v
				}
				_ = json.NewEncoder(w).Encode(store)
				return
			}
			if body["sentinel_metrics_refresh_rate_seconds"] != float64(15) {
				t.Errorf("expected sentinel_metrics_refresh_rate_seconds 15 in PATCH body, got %v", body)
				http.Error(w, `{"error":"missing sentinel_metrics_refresh_rate_seconds"}`, http.StatusBadRequest)
				return
			}
			if body["is_sentinel_debug_enabled"] != true {
				t.Errorf("expected is_sentinel_debug_enabled true in PATCH body, got %v", body)
				http.Error(w, `{"error":"missing is_sentinel_debug_enabled"}`, http.StatusBadRequest)
				return
			}
			if body["sentinel_custom_url"] != "https://sentinel.example.com" {
				t.Errorf("expected sentinel_custom_url in PATCH body, got %v", body)
				http.Error(w, `{"error":"missing sentinel_custom_url"}`, http.StatusBadRequest)
				return
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
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_sentinel" "test" {
  server_uuid                           = "` + serverUUID + `"
  is_sentinel_enabled                   = true
  is_metrics_enabled                    = true
  is_sentinel_debug_enabled             = true
  sentinel_metrics_refresh_rate_seconds = 15
  sentinel_custom_url                   = "https://sentinel.example.com"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_sentinel.test", "sentinel_metrics_refresh_rate_seconds", "15"),
				resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_sentinel_debug_enabled", "true"),
				resource.TestCheckResourceAttr("coolify_server_sentinel.test", "sentinel_custom_url", "https://sentinel.example.com"),
			),
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
