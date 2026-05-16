package application_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/spectest"
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
		EnvironmentName:    "production",
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
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
// TestDockerfileApplicationResource_CreateReadBackFailurePreservesState
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:               "dockerfile-readback-failure-uuid",
		ProjectUUID:        "aaaa0009-0009-4000-8000-000000000009",
		ServerUUID:         "bbbb0009-0009-4000-8000-000000000009",
		EnvironmentName:    "production",
		DockerfileLocation: "RlJPTSBuZ2lueA==",
		PortsExposes:       "80",
	}

	var forceReadFailure atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		forceReadFailure.Store(true)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		if forceReadFailure.Load() {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		forceReadFailure.Store(false)
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					project_uuid        = "aaaa0009-0009-4000-8000-000000000009"
					server_uuid         = "bbbb0009-0009-4000-8000-000000000009"
					dockerfile_location = "RlJPTSBuZ2lueA=="
					ports_exposes       = "80"
				`),
				ExpectError: regexp.MustCompile(`(?s)Application created but refresh failed.*Could not read application.*partial Terraform state was saved`),
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
		EnvironmentName:    "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["name"].(string); ok {
			app.Name = v
		}
		if v, ok := requestBody["description"].(string); ok {
			app.Description = v
		}
		if v, ok := requestBody["install_command"].(string); ok {
			app.InstallCommand = v
		}
		if v, ok := requestBody["build_command"].(string); ok {
			app.BuildCommand = v
		}
		if _, has := requestBody["dockerfile"]; has {
			t.Error("PATCH should not send 'dockerfile' (not accepted by Coolify v4 update API)")
			http.Error(w, `{"error":"unexpected dockerfile field"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "dockerfile_location", "/Dockerfile"),
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
	// Simulate Coolify's URL normalization: API returns bare slug for
	// git_repository. The provider should reconstruct the full URL on import.
	app := client.Application{
		UUID:               "aaaa0001-0001-4000-8000-000000000001",
		Name:               "imported-dockerfile-app",
		GitRepository:      "myorg/myapp", // Coolify strips the prefix
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:         "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName:    "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
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
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
			EnvironmentName:    "production",
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
	return acctest.TestResourceConfig(endpoint, "coolify_dockerfile_application", "test", attrs)
}
