package integrationtoken_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestIntegrationTokenResource_Lifecycle(t *testing.T) {
	t.Parallel()
	const uuid = "11111111-1111-4111-8111-111111111111"
	var mu sync.Mutex
	tokens := map[string]map[string]any{
		uuid: {
			"uuid": uuid, "name": "cf", "provider": "cloudflare",
			"capabilities": []string{"dns"}, "automatic_dns": true, "proxied_dns": false,
			"dns_zones_count": 2, "created_at": "2026-11-02T00:00:00Z", "updated_at": "2026-11-02T00:00:00Z",
		},
	}
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/security/integration-tokens", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["name"] != "cf" || body["provider"] != "cloudflare" || body["token"] != "cf-token" {
			http.Error(w, "unexpected create body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
	})
	mux.HandleFunc("GET /api/v1/security/integration-tokens/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gone := deleted
		item := tokens[uuid]
		mu.Unlock()
		if gone {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(item)
	})
	mux.HandleFunc("PATCH /api/v1/security/integration-tokens/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		item := tokens[uuid]
		if name, ok := body["name"].(string); ok {
			item["name"] = name
		}
		item["updated_at"] = "2026-11-02T01:00:00Z"
		tokens[uuid] = item
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(item)
	})
	mux.HandleFunc("DELETE /api/v1/security/integration-tokens/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Integration token deleted."})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.6.0"))
	defer srv.Close()

	config := func(name string) string {
		return acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_integration_token" "test" {
  name                 = "` + name + `"
  integration_provider = "cloudflare"
  token                = "cf-token"
}
`
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("cf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_integration_token.test", "uuid", uuid),
					resource.TestCheckResourceAttr("coolify_integration_token.test", "integration_provider", "cloudflare"),
					resource.TestCheckResourceAttr("coolify_integration_token.test", "dns_zones_count", "2"),
				),
			},
			{
				Config: config("cf-renamed"),
				Check:  resource.TestCheckResourceAttr("coolify_integration_token.test", "name", "cf-renamed"),
			},
			{
				ResourceName:                         "coolify_integration_token.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        uuid,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"token", "metadata_base_url", "metadata_client_id", "metadata_namespace", "metadata_timeout"},
			},
		},
	})
}

func TestIntegrationTokenResource_Disappears(t *testing.T) {
	t.Parallel()
	const uuid = "22222222-2222-4222-8222-222222222222"
	var mu sync.Mutex
	deleted := false
	item := map[string]any{
		"uuid": uuid, "name": "cf", "provider": "cloudflare",
		"capabilities": []string{"dns"}, "automatic_dns": true, "proxied_dns": false,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/security/integration-tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
	})
	mux.HandleFunc("GET /api/v1/security/integration-tokens/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		gone := deleted
		mu.Unlock()
		if gone {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(item)
	})
	mux.HandleFunc("DELETE /api/v1/security/integration-tokens/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.6.0"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_integration_token" "test" {
  name                 = "cf"
  integration_provider = "cloudflare"
  token                = "cf-token"
}
`,
				ExpectNonEmptyPlan: true,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["coolify_integration_token.test"]
					if !ok {
						return fmt.Errorf("resource not found in state")
					}
					if rs.Primary.Attributes["uuid"] != uuid {
						return fmt.Errorf("uuid = %s", rs.Primary.Attributes["uuid"])
					}
					mu.Lock()
					deleted = true
					mu.Unlock()
					return nil
				},
			},
		},
	})
}

func TestIntegrationTokenResource_RejectsOldCoolify(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.NewServeMux(), "4.5.3"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_integration_token" "test" {
  name                 = "cf"
  integration_provider = "cloudflare"
  token                = "cf-token"
}
`,
			ExpectError: regexp.MustCompile(`Coolify >= v4.6.0`),
		}},
	})
}
