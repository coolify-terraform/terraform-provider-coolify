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
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid", "github_app_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/private-github-app missing required field %q", field)
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_github_app", "/api/v1/applications/"),
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
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "uuid", "ghapp-app-uuid"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "name", "my-github-app"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "git_repository", "github.com/myorg/myrepo"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "git_branch", "main"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "ports_exposes", "3000"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "environment_name", "production"),
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
	deleted := false
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
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "name", "my-github-app"),
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
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "uuid", "ghapp-upd-uuid"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "name", "updated-github-app"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "description", "Updated via test"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "install_command", "yarn install"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "start_command", "yarn start"),
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
				ResourceName:                         "coolify_application_github_app.test",
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
					resource.TestCheckResourceAttrSet("coolify_application_github_app.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_application_github_app.test", "/api/v1/applications/"),
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
	mu := sync.Mutex{}
	deleted := false

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
// TestGitHubAppApplicationResource_RedeployOnUpdate
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_RedeployOnUpdate(t *testing.T) {
	t.Parallel()

	var restartCalled atomic.Bool
	mu := sync.Mutex{}
	deleted := false
	app := client.Application{
		UUID:            "ghapp-redeploy-uuid",
		Name:            "my-ghapp",
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
		if deleted {
			mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
		mu.Unlock()
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
		if v, ok := requestBody["github_app_uuid"].(string); ok {
			app.GitHubAppUUID = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
	})
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") == app.UUID {
			restartCalled.Store(true)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"deployment_uuid": "deploy-uuid-001",
				"message":         "Restart request queued.",
			})
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
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
				Config: testGitHubAppResourceConfig(srv.URL, `
					name               = "my-ghapp"
					project_uuid       = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid        = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid    = "cccc0001-0001-4000-8000-000000000001"
					git_repository     = "github.com/myorg/myrepo"
					git_branch         = "main"
					build_pack         = "nixpacks"
					ports_exposes      = "3000"
					redeploy_on_update = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_github_app.test", "github_app_uuid", "cccc0001-0001-4000-8000-000000000001"),
			},
			{
				Config: testGitHubAppResourceConfig(srv.URL, `
					name               = "my-ghapp"
					project_uuid       = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid        = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid    = "dddd0001-0001-4000-8000-000000000001"
					git_repository     = "github.com/myorg/myrepo"
					git_branch         = "main"
					build_pack         = "nixpacks"
					ports_exposes      = "3000"
					redeploy_on_update = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_github_app.test", "github_app_uuid", "dddd0001-0001-4000-8000-000000000001"),
			},
		},
	})
	if !restartCalled.Load() {
		t.Error("expected restart to be called when github_app_uuid changed with redeploy_on_update=true")
	}
}

// ---------------------------------------------------------------------------
// TestGitHubAppApplicationResource_InvalidBuildPack
// ---------------------------------------------------------------------------

func TestGitHubAppApplicationResource_InvalidBuildPack(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testGitHubAppResourceConfig(srv.URL, `
					project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
					server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
					github_app_uuid  = "dddd0001-0001-4000-8000-000000000001"
					git_repository   = "https://github.com/example/repo"
					build_pack       = "invalid_pack"
					ports_exposes    = "3000"
				`),
				ExpectError: regexp.MustCompile(`must be one of`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testGitHubAppResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_application_github_app", "test", attrs)
}
