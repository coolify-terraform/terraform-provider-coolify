package serviceapplication_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceApplicationResource_PatchAndVersionGate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version   string
		send45    bool
		sendIndex bool
		sendForce bool
		sendMax   bool
	}{
		{version: "4.5.0", send45: true, sendIndex: true, sendForce: true, sendMax: true},
		{version: "4.4.2", sendIndex: true, sendForce: true, sendMax: true},
		{version: "4.4-rc.1", sendIndex: true, sendForce: true},
		{version: "4.3.8", sendIndex: true, sendForce: true},
		{version: "4.3.1", sendIndex: true},
		{version: "4.2.0"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			var mu sync.Mutex
			var patches []map[string]any
			deleted := false
			mux := http.NewServeMux()
			app := map[string]any{
				"uuid": "bbbb0002-0002-4000-8000-000000000002", "name": "web", "fqdn": "", "status": "running",
				"max_restart_count": float64(10), "is_force_https_enabled": false,
			}
			write := func(w http.ResponseWriter) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(app)
			}
			mux.HandleFunc("GET /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, r *http.Request) {
				if deleted {
					http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
					return
				}
				write(w)
			})
			mux.HandleFunc("DELETE /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, _ *http.Request) {
				deleted = true
				http.Error(w, `{"message":"no delete"}`, http.StatusMethodNotAllowed)
			})
			mux.HandleFunc("PATCH /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var patch map[string]any
				_ = json.Unmarshal(body, &patch)
				mu.Lock()
				patches = append(patches, patch)
				for k, v := range patch {
					app[k] = v
				}
				if url, ok := patch["url"].(string); ok {
					app["fqdn"] = url
					app["url"] = url
					if len(url) > 8 && url[:8] == "https://" {
						app["url"] = url + ":443"
					}
				}
				if domains, ok := patch["noindex_domains"].([]any); ok && len(domains) > 0 {
					app["noindex_domains"] = []any{"https://app.example.com"}
				}
				mu.Unlock()
				write(w)
			})
			srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, tc.version))
			defer srv.Close()
			cfg := func(desc string, permanent bool) string {
				return acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service_application" "test" {
  service_uuid = "aaaa0001-0001-4000-8000-000000000001"
  uuid         = "bbbb0002-0002-4000-8000-000000000002"
  url          = "https://App.Example.com"
  description  = "` + desc + `"
  noindex_domains = ["https://App.Example.com"]
  is_force_https_enabled = true
  max_restart_count = 3
  is_permanent_redirect_enabled = ` + boolText(permanent) + `
  traefik_certificate_resolver  = "letsencrypt"
}
`
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: cfg("first", true), Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("coolify_service_application.test", "description", "first"),
						resource.TestCheckResourceAttr("coolify_service_application.test", "url", "https://App.Example.com"),
						resource.TestCheckResourceAttr("coolify_service_application.test", "noindex_domains.0", "https://App.Example.com"),
						resource.TestCheckResourceAttr("coolify_service_application.test", "max_restart_count", "3"),
						resource.TestCheckResourceAttr("coolify_service_application.test", "is_force_https_enabled", "true"),
					)},
					{Config: cfg("second", false), Check: resource.TestCheckResourceAttr("coolify_service_application.test", "description", "second")},
				},
			})
			mu.Lock()
			defer mu.Unlock()
			if len(patches) < 2 {
				t.Fatalf("patches = %d", len(patches))
			}
			if _, ok := patches[1]["url"]; ok {
				t.Fatalf("update resent url: %#v", patches[1])
			}
			if patches[1]["description"] != "second" {
				t.Fatalf("update description = %#v", patches[1])
			}
			_, hasRedirect := patches[0]["is_permanent_redirect_enabled"]
			_, hasResolver := patches[0]["traefik_certificate_resolver"]
			_, hasIndex := patches[0]["noindex_domains"]
			_, hasForce := patches[0]["is_force_https_enabled"]
			_, hasMax := patches[0]["max_restart_count"]
			if hasRedirect != tc.send45 || hasResolver != tc.send45 || hasIndex != tc.sendIndex || hasForce != tc.sendForce || hasMax != tc.sendMax {
				t.Fatalf("version %s redirect=%v resolver=%v index=%v force=%v max=%v body=%#v", tc.version, hasRedirect, hasResolver, hasIndex, hasForce, hasMax, patches[0])
			}
			if deleted {
				t.Fatal("destroy called DELETE")
			}
		})
	}
}

func TestServiceApplicationResource_Missing(t *testing.T) {
	t.Parallel()
	patched := false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	})
	mux.HandleFunc("PATCH /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, _ *http.Request) {
		patched = true
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service_application" "test" {
  service_uuid = "aaaa0001-0001-4000-8000-000000000001"
  uuid         = "bbbb0002-0002-4000-8000-000000000002"
  description  = "missing"
}
`,
			ExpectError: regexp.MustCompile(`Service application not found`),
		}},
	})
	if patched {
		t.Fatal("missing application was patched")
	}
}

func TestServiceApplicationResource_Import(t *testing.T) {
	t.Parallel()
	deleted := false
	mux := http.NewServeMux()
	app := map[string]any{
		"uuid": "bbbb0002-0002-4000-8000-000000000002", "name": "web", "status": "running",
	}
	mux.HandleFunc("GET /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("PATCH /api/v1/services/{uuid}/applications/{app}", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var patch map[string]any
		_ = json.Unmarshal(body, &patch)
		for k, v := range patch {
			app[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/services/{uuid}/applications/{app}", func(http.ResponseWriter, *http.Request) {
		deleted = true
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service_application" "test" {
  service_uuid = "aaaa0001-0001-4000-8000-000000000001"
  uuid         = "bbbb0002-0002-4000-8000-000000000002"
  description  = "imported"
}
`,
				Check: resource.TestCheckResourceAttr("coolify_service_application.test", "description", "imported"),
			},
			{
				ResourceName:                         "coolify_service_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001/bbbb0002-0002-4000-8000-000000000002",
				ImportStateVerifyIgnore:              []string{"force_domain_override"},
			},
		},
	})
	if deleted {
		t.Fatal("import test called DELETE")
	}
}

func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
