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
// TestGitHubAppApplicationResource_Create
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "ghapp-app-uuid",
		Name:            "my-github-app",
		GitRepository:   "github.com/myorg/myrepo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
		GitHubAppUUID:   "cccc0001-0001-4000-8000-000000000001",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_github_app_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testGitHubAppResourceConfig(srv.URL, `
					name             = "my-github-app"
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository   = "github.com/myorg/myrepo"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "uuid", "ghapp-app-uuid"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "name", "my-github-app"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "git_repository", "github.com/myorg/myrepo"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "git_branch", "main"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "ports_exposes", "3000"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "environment_name", "production"),
				),
			},
			{
				Config: testGitHubAppResourceConfig(srv.URL, `
					name             = "my-github-app"
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository   = "github.com/myorg/myrepo"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestGitHubAppApplicationResource_Update
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	app := client.Application{
		UUID:            "ghapp-upd-uuid",
		Name:            "my-github-app",
		GitRepository:   "github.com/myorg/myrepo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
		GitHubAppUUID:   "cccc0001-0001-4000-8000-000000000001",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
		if v, ok := requestBody["start_command"].(string); ok {
			app.StartCommand = v
		}
		if v, ok := requestBody["github_app_uuid"].(string); ok {
			app.GitHubAppUUID = v
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
				Config: testGitHubAppResourceConfig(srv.URL, `
					name             = "my-github-app"
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository   = "github.com/myorg/myrepo"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "name", "my-github-app"),
				),
			},
			{
				Config: testGitHubAppResourceConfig(srv.URL, `
					name             = "updated-github-app"
					description      = "Updated via test"
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository   = "github.com/myorg/myrepo"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
					install_command  = "yarn install"
					start_command    = "yarn start"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "uuid", "ghapp-upd-uuid"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "name", "updated-github-app"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "description", "Updated via test"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "install_command", "yarn install"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "start_command", "yarn start"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestGitHubAppApplicationResource_Import
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_Import(t *testing.T) {
	t.Parallel()
	// Simulate Coolify's URL normalization: API returns bare slug for
	// git_repository. The provider should reconstruct the full URL on import.
	app := client.Application{
		UUID:            "aaaa0001-0001-4000-8000-000000000001",
		Name:            "imported-ghapp",
		GitRepository:   "myorg/myrepo", // Coolify strips https://github.com/ prefix
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
		GitHubAppUUID:   "cccc0001-0001-4000-8000-000000000001",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
				// Config uses the full URL which matches normalizeGitRepository("myorg/myrepo")
				Config: testGitHubAppResourceConfig(srv.URL, `
					name             = "imported-ghapp"
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository   = "https://github.com/myorg/myrepo"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
				`),
			},
			{
				ResourceName:                         "coolify_github_app_application.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name", "github_app_uuid"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestGitHubAppApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	appUUID := "ghapp-disappear-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
			UUID:            appUUID,
			Name:            "disappearing-ghapp",
			GitRepository:   "github.com/org/repo",
			GitBranch:       "main",
			BuildPack:       "nixpacks",
			PortsExposes:    "3000",
			ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
			ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
			EnvironmentName: "production",
			GitHubAppUUID:   "cccc0001-0001-4000-8000-000000000001",
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
				Config: testGitHubAppResourceConfig(srv.URL, `
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "cccc0001-0001-4000-8000-000000000001"
					git_repository   = "github.com/org/repo"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app_application.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_github_app_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestGitHubAppApplicationResource_CreateReadBackFailurePreservesState
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()

	const appUUID = "ghapp-readback-fail-uuid"

	var forceReadFailure atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
		forceReadFailure.Store(true)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		if forceReadFailure.Load() {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Application{
			UUID:            appUUID,
			Name:            "ghapp-readback",
			GitRepository:   "github.com/org/repo",
			GitBranch:       "main",
			BuildPack:       "nixpacks",
			PortsExposes:    "3000",
			ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
			ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
			EnvironmentName: "production",
			GitHubAppUUID:   "cccc0001-0001-4000-8000-000000000001",
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testGitHubAppResourceConfig(srv.URL, `
				project_uuid    = "aaaa0001-0001-4000-8000-000000000001"
				server_uuid     = "bbbb0001-0001-4000-8000-000000000001"
				github_app_uuid = "cccc0001-0001-4000-8000-000000000001"
				git_repository  = "github.com/org/repo"
				build_pack      = "nixpacks"
				ports_exposes   = "3000"
			`),
			ExpectError: regexp.MustCompile(`(?s)Application created but refresh failed.*Could not read application.*partial Terraform state was saved`),
		}},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testGitHubAppResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_github_app_application", "test", attrs)
}
