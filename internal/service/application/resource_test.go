package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestApplicationResource_Create
// ---------------------------------------------------------------------------

func TestApplicationResource_Create(t *testing.T) {
	app := client.Application{
		UUID:          "create-app-uuid",
		Name:          "my-app",
		GitRepository: "https://github.com/example/repo",
		GitBranch:     "main",
		BuildPack:     "nixpacks",
		PortsExposes:  "3000",
		ProjectUUID:   "proj-uuid",
		ServerUUID:    "srv-uuid",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
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

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "uuid", "create-app-uuid"),
					resource.TestCheckResourceAttr("coolify_application.test", "name", "my-app"),
					resource.TestCheckResourceAttr("coolify_application.test", "git_repository", "https://github.com/example/repo"),
					resource.TestCheckResourceAttr("coolify_application.test", "git_branch", "main"),
					resource.TestCheckResourceAttr("coolify_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_application.test", "ports_exposes", "3000"),
					resource.TestCheckResourceAttr("coolify_application.test", "environment_name", "production"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_Update
// ---------------------------------------------------------------------------

func TestApplicationResource_Update(t *testing.T) {
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:          "update-app-uuid",
		Name:          "my-app",
		Description:   "initial desc",
		GitRepository: "https://github.com/example/repo",
		GitBranch:     "main",
		BuildPack:     "nixpacks",
		PortsExposes:  "3000",
		ProjectUUID:   "proj-uuid",
		ServerUUID:    "srv-uuid",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["description"].(string); ok {
			currentApp.Description = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					description    = "initial desc"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "description", "initial desc"),
				),
			},
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					description    = "updated desc"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "description", "updated desc"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_Import
// ---------------------------------------------------------------------------

func TestApplicationResource_Import(t *testing.T) {
	app := client.Application{
		UUID:          "import-app-uuid",
		Name:          "imported-app",
		GitRepository: "https://github.com/example/repo",
		GitBranch:     "main",
		BuildPack:     "nixpacks",
		PortsExposes:  "3000",
		ProjectUUID:   "proj-uuid",
		ServerUUID:    "srv-uuid",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
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

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: create the resource so it exists in state.
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
			},
			// Step 2: import and verify the imported state matches.
			{
				ResourceName:                         "coolify_application.test",
				ImportState:                          true,
				ImportStateId:                        "import-app-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_InvalidBuildPack
// ---------------------------------------------------------------------------

func TestApplicationResource_InvalidBuildPack(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
					git_repository = "https://github.com/example/repo"
					build_pack     = "invalid"
					ports_exposes  = "3000"
				`),
				ExpectError: regexp.MustCompile(`nixpacks`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testApplicationResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_application" "test" {
  %s
}
`, endpoint, attrs)
}
