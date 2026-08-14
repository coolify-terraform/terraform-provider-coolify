package cloudinitscript_test

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

func TestCloudInitScriptResource_CRUD(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/cloud-init-scripts":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] == "" || body["script"] == "" {
				t.Errorf("expected name and script, got %v", body)
			}
			body["uuid"] = "cccc0001-0001-4000-8000-000000000001"
			store[body["uuid"].(string)] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/cloud-init-scripts/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/cloud-init-scripts/")
			v, ok := store[uuid]
			if !ok {
				http.Error(w, `{}`, 404)
				return
			}
			_ = json.NewEncoder(w).Encode(v)
		case r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for _, v := range store {
				for k, val := range body {
					v[k] = val
				}
				_ = json.NewEncoder(w).Encode(v)
				return
			}
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_cloud_init_script" "test" {
  name   = "bootstrap"
  script = "#cloud-config\npackages: [nginx]\n"
}`,
				Check: resource.TestCheckResourceAttr("coolify_cloud_init_script.test", "name", "bootstrap"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_cloud_init_script" "test" {
  name   = "bootstrap2"
  script = "#cloud-config\npackages: [curl]\n"
}`,
				Check: resource.TestCheckResourceAttr("coolify_cloud_init_script.test", "name", "bootstrap2"),
			},
			{
				ResourceName:                         "coolify_cloud_init_script.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_cloud_init_script.test", "uuid"),
			},
		},
	})
}

func TestCloudInitScriptResource_Disappears(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/cloud-init-scripts":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			body["uuid"] = "cccc0001-0001-4000-8000-000000000001"
			store[body["uuid"].(string)] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/cloud-init-scripts/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/cloud-init-scripts/")
			v, ok := store[uuid]
			if !ok {
				http.Error(w, `{}`, 404)
				return
			}
			_ = json.NewEncoder(w).Encode(v)
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_cloud_init_script" "test" {
  name   = "bootstrap"
  script = "#cloud-config\npackages: [nginx]\n"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_cloud_init_script.test", "uuid"),
				acctest.CheckResourceDisappears(srv.URL, "coolify_cloud_init_script.test", "/api/v1/cloud-init-scripts/"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}
