package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerfile missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_dockerfile", "/api/v1/applications/"),
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
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "uuid", "dockerfile-app-uuid"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "name", "my-dockerfile-app"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "dockerfile_location", "/Dockerfile"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "ports_exposes", "80"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "environment_name", "production"),
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
// TestDockerfileApplicationResource_CreateRawDockerfile
// Verifies that raw (non-base64) Dockerfile content is auto-encoded before
// sending to the API. The user should not need base64encode() in HCL.
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_CreateRawDockerfile(t *testing.T) {
	t.Parallel()
	rawDockerfile := "FROM nginx:alpine\nEXPOSE 80\nCMD [\"nginx\", \"-g\", \"daemon off;\"]\n"
	app := client.Application{
		UUID:               "dockerfile-raw-uuid",
		Name:               "raw-dockerfile-app",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:         "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName:    "production",
	}

	var receivedDockerfile string
	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if df, ok := body["dockerfile"].(string); ok {
			receivedDockerfile = df
		}
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
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_dockerfile", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerfileResourceConfig(srv.URL, fmt.Sprintf(`
					name                = "raw-dockerfile-app"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = %q
					ports_exposes       = "80"
				`, rawDockerfile)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "uuid", "dockerfile-raw-uuid"),
					// State preserves user's raw content, not base64.
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "dockerfile_location", rawDockerfile),
					func(s *terraform.State) error {
						// Verify the API received base64-encoded content.
						if receivedDockerfile == "" {
							return fmt.Errorf("mock API did not receive dockerfile field")
						}
						if receivedDockerfile == rawDockerfile {
							return fmt.Errorf("expected base64-encoded content sent to API, got raw")
						}
						return nil
					},
				),
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
	mu := sync.Mutex{}
	deleted := false

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
		mu.Lock()
		defer mu.Unlock()
		if deleted {
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
		mu.Lock()
		deleted = true
		mu.Unlock()
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
	deleted := false
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
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
					name                = "my-dockerfile-app"
					project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "name", "my-dockerfile-app"),
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
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "uuid", "dockerfile-upd-uuid"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "name", "updated-dockerfile-app"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "description", "Updated via test"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "dockerfile_location", "/Dockerfile"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "build_command", "npm run build"),
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
				ResourceName:                         "coolify_application_dockerfile.test",
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
					resource.TestCheckResourceAttrSet("coolify_application_dockerfile.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_application_dockerfile.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerfileApplicationResource_RedeployOnUpdate
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_RedeployOnUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:               "dockerfile-redeploy-uuid",
		Name:               "web",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:         "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:    "production",
	}
	var restartCalled atomic.Bool
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["ports_exposes"].(string); ok {
			currentApp.PortsExposes = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, _ *http.Request) {
		restartCalled.Store(true)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Restarting."})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
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
					name                = "web"
					project_uuid        = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid         = "bbbb0002-0002-4000-8000-000000000002"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
					redeploy_on_update  = true
				`),
			},
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "web"
					project_uuid        = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid         = "bbbb0002-0002-4000-8000-000000000002"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "8080"
					redeploy_on_update  = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "ports_exposes", "8080"),
			},
		},
	})
	if !restartCalled.Load() {
		t.Error("expected restart to be called when ports_exposes changed with redeploy_on_update=true")
	}
}

// ---------------------------------------------------------------------------
// TestDockerfileApplicationResource_RedeployOnNameChange
// ---------------------------------------------------------------------------

func TestDockerfileApplicationResource_RedeployOnNameChange(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:               "dockerfile-redeploy-name-uuid",
		Name:               "original-name",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
		ProjectUUID:        "aaaa0003-0003-4000-8000-000000000003",
		ServerUUID:         "bbbb0003-0003-4000-8000-000000000003",
		EnvironmentName:    "production",
	}
	var restartCalled atomic.Bool
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["name"].(string); ok {
			currentApp.Name = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, _ *http.Request) {
		restartCalled.Store(true)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Restarting."})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
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
					name                = "original-name"
					project_uuid        = "aaaa0003-0003-4000-8000-000000000003"
					server_uuid         = "bbbb0003-0003-4000-8000-000000000003"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
					redeploy_on_update  = true
				`),
			},
			{
				Config: testDockerfileResourceConfig(srv.URL, `
					name                = "renamed-app"
					project_uuid        = "aaaa0003-0003-4000-8000-000000000003"
					server_uuid         = "bbbb0003-0003-4000-8000-000000000003"
					dockerfile_location = "/Dockerfile"
					ports_exposes       = "80"
					redeploy_on_update  = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "name", "renamed-app"),
			},
		},
	})
	if !restartCalled.Load() {
		t.Error("expected restart to be called when name changed with redeploy_on_update=true")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testDockerfileResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_application_dockerfile", "test", attrs)
}
