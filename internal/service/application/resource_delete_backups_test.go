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

func TestApplicationResource_DeleteBackupsTrue(t *testing.T) {
	t.Parallel()
	const uuid = "app-delete-backups"
	app := client.Application{
		UUID:            uuid,
		Name:            "backup-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}
	var mu sync.Mutex
	var deleteQuery string
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["delete_backups"]; exists {
			t.Errorf("create sent delete_backups")
			http.Error(w, `{"error":"delete_backups is destroy-only"}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{id}", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["delete_backups"]; exists {
			t.Errorf("update sent delete_backups")
			http.Error(w, `{"error":"delete_backups is destroy-only"}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("GET /api/v1/applications/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != uuid {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		gone := deleted
		mu.Unlock()
		if gone {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleteQuery = r.URL.RawQuery
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.8.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					delete_backups = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "delete_backups", "true"),
				),
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if deleteQuery != "delete_backups=true" {
		t.Fatalf("DELETE query = %q", deleteQuery)
	}
}
