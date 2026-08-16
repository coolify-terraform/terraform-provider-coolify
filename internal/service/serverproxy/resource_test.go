package serverproxy_test

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

func TestServerProxyResource_CRUD(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"redirect_enabled":      false,
		"redirect_url":          "",
		"generate_exact_labels": false,
		"proxy_type":            "traefik",
		"configuration":         "",
	}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/proxy/configuration") && r.Method == http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode proxy config: %v", err)
			}
			if body["configuration"] == nil {
				t.Errorf("expected configuration in PUT body, got %v", body)
			}
			store["configuration"] = body["configuration"]
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode proxy patch: %v", err)
			}
			if len(body) == 0 {
				t.Errorf("expected PATCH body, got empty")
			}
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodGet:
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
resource "coolify_server_proxy" "test" {
  server_uuid  = "` + serverUUID + `"
  proxy_type   = "caddy"
  redirect_url = "https://example.com"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "proxy_type", "caddy"),
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "redirect_url", "https://example.com"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_proxy" "test" {
  server_uuid     = "` + serverUUID + `"
  proxy_type      = "traefik"
  redirect_url    = "https://example.com"
  redirect_enabled = true
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_proxy.test", "proxy_type", "traefik"),
			},
			{
				ResourceName:                         "coolify_server_proxy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}

// Destroy leaves remote proxy configuration in place (no DELETE API).
func TestServerProxyResource_DestroyLeavesRemote(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{"proxy_type": "traefik"}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodGet:
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
			if store["proxy_type"] != "caddy" {
				return fmt.Errorf("destroy must leave remote proxy_type, got %v", store["proxy_type"])
			}
			return nil
		},
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_proxy" "test" {
  server_uuid = "` + serverUUID + `"
  proxy_type  = "caddy"
}`,
			Check: resource.TestCheckResourceAttr("coolify_server_proxy.test", "proxy_type", "caddy"),
		}},
	})
}

func TestServerProxyResource_ConfigurationPUT(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{"proxy_type": "traefik", "configuration": ""}
	var mu sync.Mutex
	var putCount int
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/proxy/configuration") && r.Method == http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode proxy config: %v", err)
			}
			cfg, _ := body["configuration"].(string)
			if cfg == "" {
				t.Errorf("expected non-empty configuration in PUT body, got %v", body)
			}
			store["configuration"] = cfg
			putCount++
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodGet:
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
resource "coolify_server_proxy" "test" {
  server_uuid   = "` + serverUUID + `"
  proxy_type    = "traefik"
  configuration = "http:\n  routers:\n    web: {}"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_proxy.test", "configuration", "http:\n  routers:\n    web: {}"),
				func(_ *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if putCount < 1 {
						return fmt.Errorf("expected PUT /proxy/configuration, got %d calls", putCount)
					}
					return nil
				},
			),
		}},
	})
}

func TestServerProxyResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch {
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
resource "coolify_server_proxy" "test" {
  server_uuid = "aaaa0001-0001-4000-8000-000000000001"
  proxy_type  = "caddy"
}`,
			ExpectError: regexp.MustCompile(`Error applying server proxy`),
		}},
	})
}

// Coolify ServerProxyController::update calls changeProxy() whenever
// proxy_type is present. That refresh can drop redirect_url from the same
// request. The provider must PATCH type first, then redirect fields.
func TestServerProxyResource_SplitTypeAndRedirect(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"redirect_url": "",
		"proxy_type":   "traefik",
	}
	var mu sync.Mutex
	var combinedPatch bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode proxy patch: %v", err)
			}
			_, hasType := body["proxy_type"]
			_, hasURL := body["redirect_url"]
			if hasType && hasURL {
				combinedPatch = true
				// Same-request changeProxy() drops the URL (Coolify tip).
				delete(body, "redirect_url")
				store["redirect_url"] = ""
			}
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodGet:
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
resource "coolify_server_proxy" "test" {
  server_uuid  = "` + serverUUID + `"
  proxy_type   = "caddy"
  redirect_url = "https://example.com"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "redirect_url", "https://example.com"),
					func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if combinedPatch {
							return fmt.Errorf("PATCH sent proxy_type and redirect_url together")
						}
						if store["redirect_url"] != "https://example.com" {
							return fmt.Errorf("store redirect_url=%v", store["redirect_url"])
						}
						return nil
					},
				),
			},
			{
				ResourceName:                         "coolify_server_proxy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}

func TestServerProxyResource_SameTypeSkipsTypePatch(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"redirect_url": "",
		"proxy_type":   "TRAEFIK",
	}
	var mu sync.Mutex
	var sawTypePatch bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode proxy patch: %v", err)
			}
			if _, ok := body["proxy_type"]; ok {
				sawTypePatch = true
			}
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodGet:
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
resource "coolify_server_proxy" "test" {
  server_uuid  = "` + serverUUID + `"
  proxy_type   = "traefik"
  redirect_url = "https://example.com"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_proxy.test", "redirect_url", "https://example.com"),
				func(_ *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if sawTypePatch {
						return fmt.Errorf("PATCH sent proxy_type when GET already had traefik")
					}
					if store["redirect_url"] != "https://example.com" {
						return fmt.Errorf("store redirect_url=%v", store["redirect_url"])
					}
					return nil
				},
			),
		}},
	})
}

func TestServerProxyResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{"proxy_type": "traefik"}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case strings.HasSuffix(r.URL.Path, "/proxy") && r.Method == http.MethodGet:
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
resource "coolify_server_proxy" "test" {
  server_uuid = "` + serverUUID + `"
  proxy_type  = "traefik"
}`,
			},
			{
				ResourceName:  "coolify_server_proxy.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}
