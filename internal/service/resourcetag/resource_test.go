package resourcetag_test

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

func TestResourceTag_Attach(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	attached := false
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tags"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["tag_name"] != "frontend" {
				t.Errorf("expected tag_name frontend, got %v", body)
			}
			attached = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/tags"):
			out := []any{}
			if attached {
				out = append(out, map[string]string{"uuid": "tttt0001-0001-4000-8000-000000000001", "name": "frontend"})
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			attached = false
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
resource "coolify_resource_tag" "test" {
  resource_type = "application"
  resource_uuid = "aaaa0001-0001-4000-8000-000000000001"
  tag_name      = "frontend"
}`,
			Check: resource.TestCheckResourceAttr("coolify_resource_tag.test", "tag_uuid", "tttt0001-0001-4000-8000-000000000001"),
		}},
	})
}

func TestResourceTag_Disappears(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	attached := false
	const (
		appUUID = "aaaa0001-0001-4000-8000-000000000001"
		tagUUID = "tttt0001-0001-4000-8000-000000000001"
	)
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tags"):
			attached = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/tags"):
			out := []any{}
			if attached {
				out = append(out, map[string]string{"uuid": tagUUID, "name": "frontend"})
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			attached = false
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
resource "coolify_resource_tag" "test" {
  resource_type = "application"
  resource_uuid = "` + appUUID + `"
  tag_name      = "frontend"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_resource_tag.test", "tag_uuid", tagUUID),
				acctest.CheckPathDisappears(srv.URL, "/api/v1/applications/"+appUUID+"/tags/"+tagUUID),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}
