package integrationtoken_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIntegrationTokenDataSource_Read(t *testing.T) {
	t.Parallel()
	const uuid = "33333333-3333-4333-8333-333333333333"
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/security/integration-tokens/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != uuid {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid": uuid, "name": "vault-read", "provider": "cloudflare",
			"capabilities": []string{"dns"}, "automatic_dns": true, "proxied_dns": false,
			"dns_zones_count": 0, "zones_synced_at": nil,
			"created_at": "2026-11-02T00:00:00Z", "updated_at": "2026-11-02T00:00:00Z",
		})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.6.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_integration_token" "test" {
  uuid = "` + uuid + `"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_integration_token.test", "name", "vault-read"),
				resource.TestCheckResourceAttr("data.coolify_integration_token.test", "integration_provider", "cloudflare"),
			),
		}},
	})
}

func TestIntegrationTokenDataSource_RejectsOldCoolify(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.NewServeMux(), "4.5.3"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_integration_token" "test" {
  uuid = "33333333-3333-4333-8333-333333333333"
}
`,
			ExpectError: regexp.MustCompile(`Coolify >= v4.6.0`),
		}},
	})
}

func TestIntegrationTokenListDataSource_Read(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/security/integration-tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"uuid": "44444444-4444-4444-8444-444444444444",
			"name": "listed", "provider": "doppler",
			"capabilities": []string{"secrets"},
		}})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.6.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_integration_tokens" "all" {}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_integration_tokens.all", "integration_tokens.#", "1"),
				resource.TestCheckResourceAttr("data.coolify_integration_tokens.all", "integration_tokens.0.name", "listed"),
				resource.TestCheckResourceAttr("data.coolify_integration_tokens.all", "integration_tokens.0.integration_provider", "doppler"),
			),
		}},
	})
}

func TestIntegrationTokenListDataSource_RejectsOldCoolify(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.NewServeMux(), "4.5.3"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_integration_tokens" "all" {}
`,
			ExpectError: regexp.MustCompile(`Coolify >= v4.6.0`),
		}},
	})
}
