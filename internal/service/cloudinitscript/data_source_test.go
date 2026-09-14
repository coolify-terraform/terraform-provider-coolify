package cloudinitscript_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudInitScriptDataSource(t *testing.T) {
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
			if body["name"] != "ds-bootstrap" {
				t.Errorf("expected POST name ds-bootstrap, got %v", body["name"])
			}
			if body["script"] == "" {
				t.Errorf("expected POST script, got %v", body)
			}
			body["uuid"] = "cccc0002-0002-4000-8000-000000000002"
			store[body["uuid"].(string)] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/cloud-init-scripts/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/cloud-init-scripts/")
			v, ok := store[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(v)
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
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
resource "coolify_cloud_init_script" "source" {
  name   = "ds-bootstrap"
  script = "#cloud-config\npackages: [nginx]\n"
}

data "coolify_cloud_init_script" "test" {
  uuid = coolify_cloud_init_script.source.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coolify_cloud_init_script.test", "uuid",
						"coolify_cloud_init_script.source", "uuid",
					),
					resource.TestCheckResourceAttr("data.coolify_cloud_init_script.test", "name", "ds-bootstrap"),
				),
			},
		},
	})
}

func TestCloudInitScriptDataSource_NotFound(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_cloud_init_script" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`(?s)Error reading cloud-init script.*00000000-0000-4000-8000-000000000000`),
			},
		},
	})
}
