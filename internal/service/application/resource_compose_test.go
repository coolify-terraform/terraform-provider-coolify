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
// TestDockerComposeApplicationResource_Create
// ---------------------------------------------------------------------------

func TestDockerComposeApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:             "compose-app-uuid",
		Name:             "my-compose-app",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
		ProjectUUID:      "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:       "bbbb0002-0002-4000-8000-000000000002",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockercompose", func(w http.ResponseWriter, r *http.Request) {
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_docker_compose_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "my-compose-app"
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "uuid", "compose-app-uuid"),
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "name", "my-compose-app"),
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "environment_name", "production"),
				),
			},
			{
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "my-compose-app"
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerComposeApplicationResource_Update
// ---------------------------------------------------------------------------

func TestDockerComposeApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:             "compose-upd-uuid",
		Name:             "my-compose-app",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
		ProjectUUID:      "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:       "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockercompose", func(w http.ResponseWriter, r *http.Request) {
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
		if v, ok := body["docker_compose_raw"].(string); ok {
			currentApp.DockerComposeRaw = v
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
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "my-compose-app"
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "docker_compose_raw", "version: '3'\nservices:\n  web:\n    image: nginx\n"),
				),
			},
			{
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "my-compose-app"
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx:1.25\n"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "docker_compose_raw", "version: '3'\nservices:\n  web:\n    image: nginx:1.25\n"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerComposeApplicationResource_Import
// ---------------------------------------------------------------------------

func TestDockerComposeApplicationResource_Import(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:             "compose-imp-uuid",
		Name:             "imported-compose-app",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
		ProjectUUID:      "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:       "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockercompose", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "imported-compose-app"
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
			},
			{
				ResourceName:                         "coolify_docker_compose_application.test",
				ImportState:                          true,
				ImportStateId:                        "compose-imp-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerComposeApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestDockerComposeApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	appUUID := "compose-disappear-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockercompose", func(w http.ResponseWriter, r *http.Request) {
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
			UUID:             appUUID,
			Name:             "disappearing-compose",
			DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
			ProjectUUID:      "aaaa0002-0002-4000-8000-000000000002",
			ServerUUID:       "bbbb0002-0002-4000-8000-000000000002",
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
				Config: testDockerComposeResourceConfig(srv.URL, `
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_docker_compose_application.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_docker_compose_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerComposeApplicationResource_Timeouts
// ---------------------------------------------------------------------------

func TestDockerComposeApplicationResource_Timeouts(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:             "compose-timeout-uuid",
		Name:             "timeout-compose",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx",
		ProjectUUID:      "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:       "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockercompose", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "timeout-compose"
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx"
					timeouts = {
						create = "30m"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "uuid", "compose-timeout-uuid"),
				),
			},
		},
	})
}

func TestDockerComposeApplicationResource_InvalidFQDN(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerComposeResourceConfig(srv.URL, `
					project_uuid       = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid        = "bbbb0002-0002-4000-8000-000000000002"
					docker_compose_raw = "version: '3'"
					fqdn               = "app.example.com"
				`),
				ExpectError: regexp.MustCompile(`must start with http:// or https://`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testDockerComposeResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_docker_compose_application", "test", attrs)
}
