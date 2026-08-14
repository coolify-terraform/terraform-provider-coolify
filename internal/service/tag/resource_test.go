package tag_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTagResource_CRUD(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/tags":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] == nil || body["name"] == "" {
				t.Errorf("expected name in create body, got %v", body)
			}
			body["uuid"] = "aaaa0001-0001-4000-8000-000000000001"
			store[body["uuid"].(string)] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/tags":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for _, v := range store {
				if name, ok := body["name"]; ok {
					v["name"] = name
				}
				_ = json.NewEncoder(w).Encode(v)
				return
			}
			http.Error(w, `{}`, 404)
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
resource "coolify_tag" "test" { name = "frontend" }`,
				Check: resource.TestCheckResourceAttr("coolify_tag.test", "name", "frontend"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_tag" "test" { name = "backend" }`,
				Check: resource.TestCheckResourceAttr("coolify_tag.test", "name", "backend"),
			},
			{
				ResourceName:                         "coolify_tag.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_tag.test", "uuid"),
			},
		},
	})
}

func TestTagResource_Disappears(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/tags":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			body["uuid"] = "aaaa0001-0001-4000-8000-000000000001"
			store[body["uuid"].(string)] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/tags":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
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
resource "coolify_tag" "test" { name = "frontend" }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_tag.test", "uuid"),
				acctest.CheckResourceDisappears(srv.URL, "coolify_tag.test", "/api/v1/tags/"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}
