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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestApplicationResource_Create
// ---------------------------------------------------------------------------

func TestApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "create-app-uuid",
		Name:            "my-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
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
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_CreateWithWebhookSecret (#575)
// ---------------------------------------------------------------------------

func TestApplicationResource_CreateWithWebhookSecret(t *testing.T) {
	t.Parallel()
	const secret = "user-github-webhook-secret"
	app := client.Application{
		UUID:            "webhook-create-uuid",
		Name:            "webhook-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0009-0009-4000-8000-000000000009",
		ServerUUID:      "bbbb0009-0009-4000-8000-000000000009",
		EnvironmentName: "production",
		// GET hides secret (no read:sensitive) unless PATCH stored it and we echo empty.
	}

	mu := sync.Mutex{}
	deleted := false
	var gotPatchSecret string

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		mu.Lock()
		if v, ok := body["manual_webhook_secret_github"].(string); ok {
			gotPatchSecret = v
			// Simulate sensitive hide: do not store on mock GET response.
			app.ManualWebhookSecretGitHub = ""
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
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
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, fmt.Sprintf(`
					project_uuid                 = "aaaa0009-0009-4000-8000-000000000009"
					server_uuid                  = "bbbb0009-0009-4000-8000-000000000009"
					git_repository               = "https://github.com/example/repo"
					build_pack                   = "nixpacks"
					ports_exposes                = "3000"
					manual_webhook_secret_github = %q
				`, secret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "uuid", "webhook-create-uuid"),
					resource.TestCheckResourceAttr("coolify_application.test", "manual_webhook_secret_github", secret),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if gotPatchSecret != secret {
							return fmt.Errorf("post-create PATCH manual_webhook_secret_github = %q, want %q", gotPatchSecret, secret)
						}
						return nil
					},
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_Update
// ---------------------------------------------------------------------------

func TestApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	currentApp := client.Application{
		UUID:            "update-app-uuid",
		Name:            "my-app",
		Description:     "initial desc",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
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
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["description"].(string); ok {
			currentApp.Description = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
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
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
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
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
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
// TestApplicationResource_UpdateReadBackFailure
// ---------------------------------------------------------------------------

func TestApplicationResource_UpdateReadBackFailure(t *testing.T) {
	t.Parallel()

	const appUUID = "update-readback-app-uuid"

	mu := sync.Mutex{}
	deleted := false
	readBackFailsAfterPatch := false
	currentApp := client.Application{
		UUID:            appUUID,
		Name:            "my-app",
		Description:     "initial desc",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
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
		if readBackFailsAfterPatch {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["description"].(string); ok {
			currentApp.Description = v
		}
		readBackFailsAfterPatch = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
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
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					description    = "initial desc"
				`),
			},
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					description    = "updated desc"
				`),
				ExpectError: regexp.MustCompile(`(?s)Error updating application.*reading application ` + appUUID + ` after update`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_Import
// ---------------------------------------------------------------------------

func TestApplicationResource_Import(t *testing.T) {
	t.Parallel()
	// Simulate Coolify's URL normalization: the API returns the stripped slug
	// (no "https://github.com/" prefix) even though the user configured the
	// full URL. The provider should reconstruct the full URL on import.
	app := client.Application{
		UUID:            "aaaa0001-0001-4000-8000-000000000001",
		Name:            "imported-app",
		GitRepository:   "example/repo", // Coolify strips the prefix
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: create the resource so it exists in state.
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
			},
			// Step 2: import and verify the imported state matches.
			{
				ResourceName:                         "coolify_application.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

func TestApplicationResource_ImportBadSimpleUUID(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "aaaa0007-0007-4000-8000-000000000007",
		Name:            "imported-app",
		GitRepository:   "example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
			},
			{
				ResourceName:  "coolify_application.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_InvalidBuildPack
// ---------------------------------------------------------------------------

func TestApplicationResource_InvalidBuildPack(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "invalid"
					ports_exposes  = "3000"
				`),
				ExpectError: regexp.MustCompile(`must be one of`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "disappear-app-uuid",
		Name:            "disappearing-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_GitRepoNormalization
// ---------------------------------------------------------------------------

func TestApplicationResource_GitRepoNormalization(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "git-norm-uuid",
		Name:            "my-app",
		GitRepository:   "example/repo", // API strips https://github.com/
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// User provides full GitHub URL, API returns stripped "example/repo"
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "git_repository", "https://github.com/example/repo"),
				),
			},
			{
				// Re-apply: should produce empty plan (no perpetual diff)
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestApplicationResource_GitRepoExternalChange verifies that when the API
// returns a different org/repo slug whose name happens to be a suffix of the
// user's configured URL (e.g. "org/repo" is a suffix of "myorg/repo"), the
// provider detects the change instead of falsely preserving the old URL.
// This is the regression test for #167 (HasSuffix imprecision).
func TestApplicationResource_GitRepoExternalChange(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	currentRepo := "myexample/repo" // initial API value (matches user config)
	app := client.Application{
		UUID:            "git-change-uuid",
		Name:            "my-app",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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
		a := app
		a.GitRepository = currentRepo
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["git_repository"]; !exists {
			t.Error("PATCH body missing git_repository field")
		}
		mu.Lock()
		repo := currentRepo
		mu.Unlock()
		a := app
		a.GitRepository = repo
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a)
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
				// Initial create: user configures myexample/repo, API returns myexample/repo
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/myexample/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "git_repository", "https://github.com/myexample/repo"),
				),
			},
			{
				// Simulate external change: API now returns "example/repo" (different org).
				// "example/repo" IS a suffix of "https://github.com/myexample/repo"
				// so the old HasSuffix check would have falsely preserved the URL.
				PreConfig: func() {
					mu.Lock()
					currentRepo = "example/repo"
					mu.Unlock()
				},
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "git_repository", "https://github.com/example/repo"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_LimitsAndHealthChecks
// ---------------------------------------------------------------------------

func TestApplicationResource_LimitsAndHealthChecks(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false

	swappiness := int64(60)
	cpuShares := int64(1024)
	hcEnabled := true
	hcInterval := int64(30)
	hcTimeout := int64(10)
	hcRetries := int64(3)
	hcStartPeriod := int64(5)
	autoDeployEnabled := true

	currentApp := client.Application{
		UUID:            "limits-hc-uuid",
		Name:            "limits-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
		// Resource limits
		LimitsMemory:            "512m",
		LimitsMemorySwap:        "1g",
		LimitsMemorySwappiness:  &swappiness,
		LimitsMemoryReservation: "256m",
		LimitsCPUs:              "0.5",
		LimitsCPUSet:            "0-1",
		LimitsCPUShares:         &cpuShares,
		// Health checks
		HealthCheckEnabled:     &hcEnabled,
		HealthCheckPath:        "/health",
		HealthCheckPort:        "3000",
		HealthCheckInterval:    &hcInterval,
		HealthCheckTimeout:     &hcTimeout,
		HealthCheckRetries:     &hcRetries,
		HealthCheckStartPeriod: &hcStartPeriod,
		// Auto-deploy
		IsAutoDeployEnabled: &autoDeployEnabled,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
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
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["limits_memory"].(string); ok {
			currentApp.LimitsMemory = v
		}
		if v, ok := requestBody["limits_memory_swap"].(string); ok {
			currentApp.LimitsMemorySwap = v
		}
		if v, ok := requestBody["limits_memory_swappiness"].(float64); ok {
			i := int64(v)
			currentApp.LimitsMemorySwappiness = &i
		}
		if v, ok := requestBody["limits_memory_reservation"].(string); ok {
			currentApp.LimitsMemoryReservation = v
		}
		if v, ok := requestBody["limits_cpus"].(string); ok {
			currentApp.LimitsCPUs = v
		}
		if v, ok := requestBody["limits_cpuset"].(string); ok {
			currentApp.LimitsCPUSet = v
		}
		if v, ok := requestBody["limits_cpu_shares"].(float64); ok {
			i := int64(v)
			currentApp.LimitsCPUShares = &i
		}
		if v, ok := requestBody["health_check_enabled"].(bool); ok {
			currentApp.HealthCheckEnabled = &v
		}
		if v, ok := requestBody["health_check_path"].(string); ok {
			currentApp.HealthCheckPath = v
		}
		if v, ok := requestBody["health_check_port"].(string); ok {
			currentApp.HealthCheckPort = v
		}
		if v, ok := requestBody["health_check_interval"].(float64); ok {
			i := int64(v)
			currentApp.HealthCheckInterval = &i
		}
		if v, ok := requestBody["health_check_timeout"].(float64); ok {
			i := int64(v)
			currentApp.HealthCheckTimeout = &i
		}
		if v, ok := requestBody["health_check_retries"].(float64); ok {
			i := int64(v)
			currentApp.HealthCheckRetries = &i
		}
		if v, ok := requestBody["health_check_start_period"].(float64); ok {
			i := int64(v)
			currentApp.HealthCheckStartPeriod = &i
		}
		if v, ok := requestBody["is_auto_deploy_enabled"].(bool); ok {
			currentApp.IsAutoDeployEnabled = &v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
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
			// Step 1: Create with resource limits + health checks + auto-deploy.
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid             = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid              = "bbbb0002-0002-4000-8000-000000000002"
					git_repository           = "https://github.com/example/repo"
					build_pack               = "nixpacks"
					ports_exposes            = "3000"
					limits_memory            = "512m"
					limits_memory_swap       = "1g"
					limits_memory_swappiness = 60
					limits_memory_reservation = "256m"
					limits_cpus              = "0.5"
					limits_cpuset            = "0-1"
					limits_cpu_shares        = 1024
					health_check_enabled     = true
					health_check_path        = "/health"
					health_check_port        = "3000"
					health_check_interval    = 30
					health_check_timeout     = 10
					health_check_retries     = 3
					health_check_start_period = 5
					is_auto_deploy_enabled   = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "uuid", "limits-hc-uuid"),
					// Resource limits
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory", "512m"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory_swap", "1g"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory_swappiness", "60"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory_reservation", "256m"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_cpus", "0.5"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_cpuset", "0-1"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_cpu_shares", "1024"),
					// Health checks
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_path", "/health"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_port", "3000"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_interval", "30"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_retries", "3"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_start_period", "5"),
					// Auto-deploy
					resource.TestCheckResourceAttr("coolify_application.test", "is_auto_deploy_enabled", "true"),
				),
			},
			// Step 2: Update some fields and verify the new values.
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid             = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid              = "bbbb0002-0002-4000-8000-000000000002"
					git_repository           = "https://github.com/example/repo"
					build_pack               = "nixpacks"
					ports_exposes            = "3000"
					limits_memory            = "1g"
					limits_memory_swap       = "2g"
					limits_memory_swappiness = 80
					limits_memory_reservation = "512m"
					limits_cpus              = "2"
					limits_cpuset            = "0-3"
					limits_cpu_shares        = 512
					health_check_enabled     = false
					health_check_path        = "/ready"
					health_check_port        = "8080"
					health_check_interval    = 60
					health_check_timeout     = 20
					health_check_retries     = 5
					health_check_start_period = 10
					is_auto_deploy_enabled   = false
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Resource limits — updated values
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory", "1g"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory_swap", "2g"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory_swappiness", "80"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_memory_reservation", "512m"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_cpus", "2"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_cpuset", "0-3"),
					resource.TestCheckResourceAttr("coolify_application.test", "limits_cpu_shares", "512"),
					// Health checks — updated values
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_path", "/ready"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_port", "8080"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_interval", "60"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_timeout", "20"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_retries", "5"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_start_period", "10"),
					// Auto-deploy — updated value
					resource.TestCheckResourceAttr("coolify_application.test", "is_auto_deploy_enabled", "false"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestApplicationResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()

	const createdAppUUID = "create-readback-404-uuid"

	var deleteCalledForCreatedUUID atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": createdAppUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != createdAppUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		deleteCalledForCreatedUUID.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(*terraform.State) error {
			if !deleteCalledForCreatedUUID.Load() {
				return fmt.Errorf("expected destroy to delete partially tracked application %s", createdAppUUID)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				ExpectError: regexp.MustCompile(`(?s)Application created but refresh failed.*partial Terraform state was saved.*becomes readable through the API`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_RedeployOnUpdate
// ---------------------------------------------------------------------------

func TestApplicationResource_RedeployOnUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:            "app-redeploy-uuid",
		Name:            "web",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
	}
	var restartCalled atomic.Bool
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
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
				Config: testApplicationResourceConfig(srv.URL, `
					name           = "web"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					git_branch     = "main"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					redeploy_on_update = true
				`),
			},
			{
				Config: testApplicationResourceConfig(srv.URL, `
					name           = "web"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					git_branch     = "main"
					build_pack     = "nixpacks"
					ports_exposes  = "8080"
					redeploy_on_update = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_application.test", "ports_exposes", "8080"),
			},
		},
	})
	if !restartCalled.Load() {
		t.Error("expected restart to be called when ports_exposes changed with redeploy_on_update=true")
	}
}

// ---------------------------------------------------------------------------
// TestApplicationResource_CreateAPIError
// ---------------------------------------------------------------------------

func TestApplicationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				ExpectError: regexp.MustCompile(`Error creating application`),
			},
		},
	})
}

func testApplicationResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_application", "test", attrs)
}

func decodeRequestBodyMap(t *testing.T, w http.ResponseWriter, r *http.Request) (map[string]interface{}, bool) {
	t.Helper()

	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		t.Errorf("decoding %s %s request body: %v", r.Method, r.URL.Path, err)
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return nil, false
	}
	return requestBody, true
}

func TestApplicationResource_InvalidPortsExposes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "abc"
				`),
				ExpectError: regexp.MustCompile(`comma-separated list of port numbers`),
			},
		},
	})
}

func TestApplicationResource_InvalidDomains(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					domains = "app.example.com"
				`),
				ExpectError: regexp.MustCompile(`must be a valid URL starting with http:// or https://`),
			},
		},
	})
}

func TestApplicationResource_InvalidUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "not-a-valid-uuid"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
				ExpectError: acctest.UUIDValidationError(),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ExtendedFields
// ---------------------------------------------------------------------------

func TestApplicationResource_ExtendedFields(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false

	isStatic := true
	isForceHTTPS := false
	hcReturnCode := int64(200)
	hcInterval := int64(5)
	hcTimeout := int64(5)
	hcRetries := int64(10)
	hcStartPeriod := int64(5)
	autoDeployEnabled := true
	hcEnabled := false
	isSPA := false
	isHTTPAuth := false
	connectDocker := false
	containerEscape := true
	preserveRepo := false
	useBuild := false

	currentApp := client.Application{
		UUID:            "ext-fields-uuid",
		Name:            "ext-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName: "production",
		// Extended fields under test
		Redirect:               "www",
		BaseDirectory:          "/app",
		HealthCheckType:        "cmd",
		HealthCheckCommand:     "curl localhost",
		HealthCheckHost:        "localhost",
		HealthCheckMethod:      "GET",
		HealthCheckScheme:      "http",
		HealthCheckReturnCode:  &hcReturnCode,
		IsStatic:               &isStatic,
		IsForceHTTPSEnabled:    &isForceHTTPS,
		PreDeploymentCommand:   "npm run migrate",
		CustomDockerRunOptions: "--memory=512m",
		StaticImage:            "nginx:alpine",
		// Computed+Default health check fields (API returns these)
		HealthCheckPath:        "/",
		HealthCheckInterval:    &hcInterval,
		HealthCheckTimeout:     &hcTimeout,
		HealthCheckRetries:     &hcRetries,
		HealthCheckStartPeriod: &hcStartPeriod,
		// Computed+Default bools (API returns these)
		IsSPA:                         &isSPA,
		IsHTTPBasicAuthEnabled:        &isHTTPAuth,
		ConnectToDockerNetwork:        &connectDocker,
		IsContainerLabelEscapeEnabled: &containerEscape,
		IsPreserveRepositoryEnabled:   &preserveRepo,
		UseBuildServer:                &useBuild,
		IsAutoDeployEnabled:           &autoDeployEnabled,
		HealthCheckEnabled:            &hcEnabled,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
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
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["redirect"].(string); ok {
			currentApp.Redirect = v
		}
		if v, ok := requestBody["base_directory"].(string); ok {
			currentApp.BaseDirectory = v
		}
		if v, ok := requestBody["is_static"].(bool); ok {
			currentApp.IsStatic = &v
		}
		if v, ok := requestBody["pre_deployment_command"].(string); ok {
			currentApp.PreDeploymentCommand = v
		}
		if v, ok := requestBody["health_check_type"].(string); ok {
			currentApp.HealthCheckType = v
		}
		if v, ok := requestBody["health_check_command"].(string); ok {
			currentApp.HealthCheckCommand = v
		}
		if v, ok := requestBody["health_check_host"].(string); ok {
			currentApp.HealthCheckHost = v
		}
		if v, ok := requestBody["health_check_method"].(string); ok {
			currentApp.HealthCheckMethod = v
		}
		if v, ok := requestBody["health_check_scheme"].(string); ok {
			currentApp.HealthCheckScheme = v
		}
		if v, ok := requestBody["health_check_return_code"].(float64); ok {
			n := int64(v)
			currentApp.HealthCheckReturnCode = &n
		}
		if v, ok := requestBody["is_force_https_enabled"].(bool); ok {
			currentApp.IsForceHTTPSEnabled = &v
		}
		if v, ok := requestBody["custom_docker_run_options"].(string); ok {
			currentApp.CustomDockerRunOptions = v
		}
		if v, ok := requestBody["static_image"].(string); ok {
			currentApp.StaticImage = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
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
			// Step 1: Create with extended fields.
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid              = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid               = "bbbb0002-0002-4000-8000-000000000002"
					git_repository            = "https://github.com/example/repo"
					build_pack                = "nixpacks"
					ports_exposes             = "3000"
					redirect                  = "www"
					base_directory            = "/app"
					health_check_type         = "cmd"
					health_check_command      = "curl localhost"
					is_static                 = true
					is_force_https_enabled    = false
					pre_deployment_command    = "npm run migrate"
					custom_docker_run_options = "--memory=512m"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "redirect", "www"),
					resource.TestCheckResourceAttr("coolify_application.test", "base_directory", "/app"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_type", "cmd"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_command", "curl localhost"),
					resource.TestCheckResourceAttr("coolify_application.test", "is_static", "true"),
					resource.TestCheckResourceAttr("coolify_application.test", "is_force_https_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_application.test", "pre_deployment_command", "npm run migrate"),
					resource.TestCheckResourceAttr("coolify_application.test", "custom_docker_run_options", "--memory=512m"),
					// Verify computed defaults are populated
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_host", "localhost"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_method", "GET"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_scheme", "http"),
					resource.TestCheckResourceAttr("coolify_application.test", "health_check_return_code", "200"),
					resource.TestCheckResourceAttr("coolify_application.test", "static_image", "nginx:alpine"),
				),
			},
			// Step 2: Update several extended fields.
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid              = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid               = "bbbb0002-0002-4000-8000-000000000002"
					git_repository            = "https://github.com/example/repo"
					build_pack                = "nixpacks"
					ports_exposes             = "3000"
					redirect                  = "non-www"
					base_directory            = "/src"
					health_check_type         = "cmd"
					health_check_command      = "curl localhost"
					is_static                 = false
					is_force_https_enabled    = false
					pre_deployment_command    = "npm run seed"
					custom_docker_run_options = "--memory=512m"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "redirect", "non-www"),
					resource.TestCheckResourceAttr("coolify_application.test", "base_directory", "/src"),
					resource.TestCheckResourceAttr("coolify_application.test", "is_static", "false"),
					resource.TestCheckResourceAttr("coolify_application.test", "pre_deployment_command", "npm run seed"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ValidateRedirect
// ---------------------------------------------------------------------------

func TestApplicationResource_ValidateRedirect(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					redirect       = "invalid"
				`),
				ExpectError: regexp.MustCompile(`www`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ValidatePortsMappings
// ---------------------------------------------------------------------------

func TestApplicationResource_ValidatePortsMappings(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					ports_mappings = "abc"
				`),
				ExpectError: regexp.MustCompile(`host:container`),
			},
		},
	})
}

func TestApplicationResource_PortsMappingsRange(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
					ports_mappings = "99999:80"
				`),
				ExpectError: regexp.MustCompile(`between 1 and 65535`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ImportCompound
// ---------------------------------------------------------------------------

func TestApplicationResource_ImportCompound(t *testing.T) {
	t.Parallel()
	const (
		projUUID = "aaaa0004-0004-4000-8000-000000000004"
		srvUUID  = "bbbb0004-0004-4000-8000-000000000004"
		appUUID  = "cccc0004-0004-4000-8000-000000000004"
		envName  = "production"
	)

	app := client.Application{
		UUID:            appUUID,
		Name:            "compound-import-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     projUUID,
		ServerUUID:      srvUUID,
		EnvironmentName: envName,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
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
		json.NewEncoder(w).Encode(app)
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
	mux.HandleFunc("GET /api/v1/servers/{uuid}/resources", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != srvUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"uuid": appUUID, "name": app.Name, "type": "application"},
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, fmt.Sprintf(`
					project_uuid   = %q
					server_uuid    = %q
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`, projUUID, srvUUID)),
			},
			{
				ResourceName:  "coolify_application.test",
				ImportState:   true,
				ImportStateId: projUUID + ":" + srvUUID + ":" + envName + ":" + appUUID,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(states))
					}
					attrs := states[0].Attributes
					checks := map[string]string{
						"project_uuid":     projUUID,
						"server_uuid":      srvUUID,
						"environment_name": envName,
						"uuid":             appUUID,
					}
					for k, want := range checks {
						if got := attrs[k]; got != want {
							return fmt.Errorf("attribute %s = %q, want %q", k, got, want)
						}
					}
					return nil
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ImportCompoundWrongServer
// ---------------------------------------------------------------------------

func TestApplicationResource_ImportCompoundWrongServer(t *testing.T) {
	t.Parallel()
	const (
		projUUID     = "aaaa0007-0007-4000-8000-000000000007"
		srvUUID      = "bbbb0007-0007-4000-8000-000000000007"
		wrongSrvUUID = "bbbb0008-0008-4000-8000-000000000008"
		appUUID      = "cccc0007-0007-4000-8000-000000000007"
		envName      = "production"
	)

	app := client.Application{
		UUID:            appUUID,
		Name:            "wrong-server-import-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     projUUID,
		ServerUUID:      srvUUID,
		EnvironmentName: envName,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	// Wrong server lists no matching application UUID.
	mux.HandleFunc("GET /api/v1/servers/{uuid}/resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.PathValue("uuid") == srvUUID {
			json.NewEncoder(w).Encode([]map[string]string{
				{"uuid": appUUID, "name": app.Name, "type": "application"},
			})
			return
		}
		json.NewEncoder(w).Encode([]map[string]string{})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testApplicationResourceConfig(srv.URL, fmt.Sprintf(`
					project_uuid   = %q
					server_uuid    = %q
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`, projUUID, srvUUID)),
			},
			{
				ResourceName:  "coolify_application.test",
				ImportState:   true,
				ImportStateId: projUUID + ":" + wrongSrvUUID + ":" + envName + ":" + appUUID,
				ExpectError:   regexp.MustCompile(`is not deployed on server`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ImportCompoundBadParts
// ---------------------------------------------------------------------------

func TestApplicationResource_ImportCompoundBadParts(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "cccc0005-0005-4000-8000-000000000005",
		Name:            "bad-parts-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0005-0005-4000-8000-000000000005",
		ServerUUID:      "bbbb0005-0005-4000-8000-000000000005",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0005-0005-4000-8000-000000000005"
					server_uuid    = "bbbb0005-0005-4000-8000-000000000005"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
			},
			{
				ResourceName:  "coolify_application.test",
				ImportState:   true,
				ImportStateId: "a:b:c",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestApplicationResource_ImportCompoundEmptyEnv
// ---------------------------------------------------------------------------

func TestApplicationResource_ImportCompoundEmptyEnv(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:            "cccc0006-0006-4000-8000-000000000006",
		Name:            "empty-env-app",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0006-0006-4000-8000-000000000006",
		ServerUUID:      "bbbb0006-0006-4000-8000-000000000006",
		EnvironmentName: "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/public missing required field %q", field)
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
				Config: testApplicationResourceConfig(srv.URL, `
					project_uuid   = "aaaa0006-0006-4000-8000-000000000006"
					server_uuid    = "bbbb0006-0006-4000-8000-000000000006"
					git_repository = "https://github.com/example/repo"
					build_pack     = "nixpacks"
					ports_exposes  = "3000"
				`),
			},
			{
				ResourceName:  "coolify_application.test",
				ImportState:   true,
				ImportStateId: "aaaa0006-0006-4000-8000-000000000006:bbbb0006-0006-4000-8000-000000000006::cccc0006-0006-4000-8000-000000000006",
				ExpectError:   regexp.MustCompile(`environment_name must not be empty`),
			},
		},
	})
}
