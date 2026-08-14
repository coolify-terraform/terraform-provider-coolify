package gitlabapp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGitLabAppResource_CRUD(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/gitlab-apps":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] == "" || body["html_url"] == "" {
				t.Errorf("expected name and html_url, got %v", body)
			}
			body["id"] = float64(7)
			body["uuid"] = "gggg0001-0001-4000-8000-000000000001"
			if body["api_url"] == nil {
				body["api_url"] = body["html_url"].(string) + "/api/v4"
			}
			store["7"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gitlab-apps":
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
resource "coolify_gitlab_app" "test" {
  name     = "corp-gitlab"
  html_url = "https://gitlab.example.com"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "name", "corp-gitlab"),
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "id", "7"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name     = "corp-gitlab-2"
  html_url = "https://gitlab.example.com"
}`,
				Check: resource.TestCheckResourceAttr("coolify_gitlab_app.test", "name", "corp-gitlab-2"),
			},
		},
	})
}
