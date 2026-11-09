//go:build !ci_app_b

package application_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGiteaAppApplicationResource_CreateSendsGiteaUUID(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "gitea-app-uuid",
		Name:            "my-gitea-app",
		GitRepository:   "gitea.example.com/org/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
	}
	gitea := "cccc0001-0001-4000-8000-000000000001"

	var mu sync.Mutex
	deleted := false
	var patched map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-gitea-app", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if body["gitea_app_uuid"] != gitea {
			t.Errorf("gitea_app_uuid = %v", body["gitea_app_uuid"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		mu.Lock()
		patched = body
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_gitea_app", "test", `
					name            = "my-gitea-app"
					project_uuid    = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid     = "bbbb0001-0001-4000-8000-000000000001"
					gitea_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository  = "gitea.example.com/org/repo"
					git_branch      = "main"
					build_pack      = "nixpacks"
					ports_exposes   = "3000"
					is_image_scan_enabled = false
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "uuid", "gitea-app-uuid"),
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "gitea_app_uuid", gitea),
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "is_image_scan_enabled", "false"),
				),
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if patched == nil {
		t.Fatal("post-create PATCH was not sent")
	}
	if patched["is_image_scan_enabled"] != false {
		t.Fatalf("post-create PATCH is_image_scan_enabled = %v, want false", patched["is_image_scan_enabled"])
	}
}
