package sharedenv_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSharedEnvResource_TeamCRUD(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/team/envs":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["key"] != "GLOBAL_FLAG" {
				t.Errorf("expected key GLOBAL_FLAG got %v", body)
			}
			body["id"] = float64(3)
			store["3"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/team/envs":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
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
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "on"
}`,
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", "GLOBAL_FLAG"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "off"
}`,
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
		},
	})
}
