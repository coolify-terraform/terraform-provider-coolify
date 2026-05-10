package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestDockerfileApplicationResource_Create
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:               "dockerfile-app-uuid",
		Name:               "my-dockerfile-app",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:         "bbbb0001-0001-4000-8000-000000000001",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(spectest.WithSpecAudit(t, "coolify-v4",
		acctest.WithVersionEndpoint(mux)))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_dockerfile_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "my-dockerfile-app"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "uuid", "dockerfile-app-uuid"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "name", "my-dockerfile-app"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "dockerfile_location", "/Dockerfile"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "ports_exposes", "80"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "environment_name", "production"),
				),
			},
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "my-dockerfile-app"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerfileApplicationResource_Update
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	app := client.Application{
		UUID:               "dockerfile-upd-uuid",
		Name:               "my-dockerfile-app",
		Description:        "",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:         "bbbb0001-0001-4000-8000-000000000001",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["name"].(string); ok {
			app.Name = v
		}
		if v, ok := body["description"].(string); ok {
			app.Description = v
		}
		if v, ok := body["install_command"].(string); ok {
			app.InstallCommand = v
		}
		if v, ok := body["build_command"].(string); ok {
			app.BuildCommand = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "my-dockerfile-app"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "name", "my-dockerfile-app"),
				),
			},
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "updated-dockerfile-app"
					description         = "Updated via test"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
					install_command     = "npm install"
					build_command       = "npm run build"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "uuid", "dockerfile-upd-uuid"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "name", "updated-dockerfile-app"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "description", "Updated via test"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "build_command", "npm run build"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerfileApplicationResource_Import
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_Import(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:               "dockerfile-imp-uuid",
		Name:               "imported-dockerfile-app",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:         "bbbb0001-0001-4000-8000-000000000001",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "imported-dockerfile-app"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
				`),
			},
			{
				ResourceName:                         "coolify_dockerfile_application.test",
				ImportState:                          true,
				ImportStateId:                        "dockerfile-imp-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerfileApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	appUUID := "dockerfile-disappear-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Application{
			UUID:               appUUID,
			Name:               "disappearing-dockerfile",
			DockerfileLocation: "/Dockerfile",
			PortsExposes:       "80",
			ProjectUUID:        "aaaa0001-0001-4000-8000-000000000001",
			ServerUUID:         "bbbb0001-0001-4000-8000-000000000001",
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_dockerfile_application.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_dockerfile_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testDockerfileResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_dockerfile_application" "test" {
  %s
}
`, endpoint, attrs)
}
