package application_test

import (
	"encoding/json"
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
// TestPrivateGitApplicationResource_Create
// ---------------------------------------------------------------------------

func TestPrivateGitApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:          "pgit-app-uuid",
		Name:          "api-server",
		GitRepository: "git@github.com:myorg/api-server.git",
		GitBranch:     "main",
		BuildPack:     "dockerfile",
		PortsExposes:  "8080",
		ProjectUUID:   "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:    "bbbb0002-0002-4000-8000-000000000002",
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_private_git_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					name             = "api-server"
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:myorg/api-server.git"
					git_branch       = "main"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					build_pack       = "dockerfile"
					ports_exposes    = "8080"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "uuid", "pgit-app-uuid"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "name", "api-server"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "git_repository", "git@github.com:myorg/api-server.git"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "git_branch", "main"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "build_pack", "dockerfile"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "ports_exposes", "8080"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "environment_name", "production"),
				),
			},
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					name             = "api-server"
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:myorg/api-server.git"
					git_branch       = "main"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					build_pack       = "dockerfile"
					ports_exposes    = "8080"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPrivateGitApplicationResource_Update
// ---------------------------------------------------------------------------

func TestPrivateGitApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:          "pgit-upd-uuid",
		Name:          "api-server",
		Description:   "initial desc",
		GitRepository: "git@github.com:myorg/api-server.git",
		GitBranch:     "main",
		BuildPack:     "dockerfile",
		PortsExposes:  "8080",
		ProjectUUID:   "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:    "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					name             = "api-server"
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:myorg/api-server.git"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					build_pack       = "dockerfile"
					ports_exposes    = "8080"
					description      = "initial desc"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "description", "initial desc"),
				),
			},
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					name             = "api-server"
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:myorg/api-server.git"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					build_pack       = "dockerfile"
					ports_exposes    = "8080"
					description      = "updated desc"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "description", "updated desc"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPrivateGitApplicationResource_Import
// ---------------------------------------------------------------------------

func TestPrivateGitApplicationResource_Import(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:          "pgit-imp-uuid",
		Name:          "imported-pgit-app",
		GitRepository: "git@github.com:myorg/api-server.git",
		GitBranch:     "main",
		BuildPack:     "dockerfile",
		PortsExposes:  "8080",
		ProjectUUID:   "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:    "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testPrivateGitResourceConfig(srv.URL, `
					name             = "imported-pgit-app"
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:myorg/api-server.git"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					build_pack       = "dockerfile"
					ports_exposes    = "8080"
				`),
			},
			{
				ResourceName:                         "coolify_private_git_application.test",
				ImportState:                          true,
				ImportStateId:                        "pgit-imp-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name", "private_key_uuid"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPrivateGitApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestPrivateGitApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	appUUID := "pgit-disappear-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
			UUID:          appUUID,
			Name:          "disappearing-pgit",
			GitRepository: "git@github.com:org/repo.git",
			GitBranch:     "main",
			BuildPack:     "nixpacks",
			PortsExposes:  "3000",
			ProjectUUID:   "aaaa0002-0002-4000-8000-000000000002",
			ServerUUID:    "bbbb0002-0002-4000-8000-000000000002",
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
				Config: testPrivateGitResourceConfig(srv.URL, `
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:org/repo.git"
					git_branch       = "main"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
					private_key_uuid = "dddd0005-0005-4000-8000-000000000005"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_git_application.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_private_git_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPrivateGitApplicationResource_Timeouts
// ---------------------------------------------------------------------------

func TestPrivateGitApplicationResource_Timeouts(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:          "privgit-timeout-uuid",
		Name:          "timeout-privgit",
		GitRepository: "git@github.com:org/repo.git",
		GitBranch:     "main",
		BuildPack:     "nixpacks",
		PortsExposes:  "3000",
		ProjectUUID:   "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:    "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/private-github-app", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testPrivateGitResourceConfig(srv.URL, `
					name             = "timeout-privgit"
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					git_repository   = "git@github.com:org/repo.git"
					private_key_uuid = "dddd0006-0006-4000-8000-000000000006"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
					timeouts = {
						create = "30m"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "uuid", "privgit-timeout-uuid"),
				),
			},
		},
	})
}

func TestPrivateGitApplicationResource_InvalidPortsExposes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					git_repository   = "git@github.com:example/repo.git"
					build_pack       = "nixpacks"
					ports_exposes    = "abc"
				`),
				ExpectError: regexp.MustCompile(`comma-separated list of port numbers`),
			},
		},
	})
}

func TestPrivateGitApplicationResource_InvalidFQDN(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					git_repository   = "git@github.com:example/repo.git"
					build_pack       = "nixpacks"
					ports_exposes    = "3000"
					fqdn             = "app.example.com"
				`),
				ExpectError: regexp.MustCompile(`must start with http:// or https://`),
			},
		},
	})
}

func TestPrivateGitApplicationResource_InvalidBuildPack(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testPrivateGitResourceConfig(srv.URL, `
					project_uuid     = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid      = "bbbb0002-0002-4000-8000-000000000002"
					private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
					git_repository   = "git@github.com:example/repo.git"
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

func testPrivateGitResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_private_git_application", "test", attrs)
}
