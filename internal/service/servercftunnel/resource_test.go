package servercftunnel_test

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

func TestServerCFTunnelResource_CRUD(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	enabled := false
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cloudflare-tunnel/enable"):
			enabled = true
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cloudflare-tunnel/disable"):
			enabled = false
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/cloudflare-tunnel"):
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode cf tunnel patch: %v", err)
			}
			if _, ok := body["is_cloudflare_tunnel"]; !ok {
				t.Errorf("expected is_cloudflare_tunnel in PATCH body, got %v", body)
			}
			if v, ok := body["is_cloudflare_tunnel"].(bool); ok {
				enabled = v
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"is_cloudflare_tunnel": enabled})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/cloudflare-tunnel"):
			_ = json.NewEncoder(w).Encode(map[string]any{"is_cloudflare_tunnel": enabled})
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
resource "coolify_server_cloudflare_tunnel" "test" {
  server_uuid          = "` + serverUUID + `"
  is_cloudflare_tunnel = true
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_cloudflare_tunnel.test", "is_cloudflare_tunnel", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_cloudflare_tunnel" "test" {
  server_uuid          = "` + serverUUID + `"
  is_cloudflare_tunnel = false
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_cloudflare_tunnel.test", "is_cloudflare_tunnel", "false"),
			},
			{
				ResourceName:                         "coolify_server_cloudflare_tunnel.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}
