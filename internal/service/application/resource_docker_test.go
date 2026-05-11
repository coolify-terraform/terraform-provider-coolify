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
// TestDockerImageApplicationResource_Create
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-app-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_docker_image_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "uuid", "docker-app-uuid"),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "name", "nginx-proxy"),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "docker_image", "nginx:latest"),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "ports_exposes", "80"),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "environment_name", "production"),
				),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Update
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:                    "docker-upd-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
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
		if v, ok := body["docker_registry_image_name"].(string); ok {
			currentApp.DockerRegistryImageName = v
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
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "docker_image", "nginx:latest"),
				),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:1.25"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "docker_image", "nginx:1.25"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Import
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Import(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "aaaa0001-0001-4000-8000-000000000001",
		Name:                    "imported-docker-app",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "imported-docker-app"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
			},
			{
				ResourceName:                         "coolify_docker_image_application.test",
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
// TestDockerImageApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	appUUID := "docker-disappear-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
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
			UUID:                    appUUID,
			Name:                    "disappearing-docker",
			DockerRegistryImageName: "nginx:latest",
			PortsExposes:            "80",
			ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
			ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
			EnvironmentName:         "production",
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
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid  = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid   = "bbbb0002-0002-4000-8000-000000000002"
					docker_image  = "nginx:latest"
					ports_exposes = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_docker_image_application.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_docker_image_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Status
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Status(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-status-uuid",
		Name:                    "status-app",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
		Status:                  "running:healthy",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "status-app"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "status", "running:healthy"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Timeouts
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Timeouts(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-timeout-uuid",
		Name:                    "timeout-app",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
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
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "timeout-app"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
					timeouts = {
						create = "30m"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "uuid", "docker-timeout-uuid"),
				),
			},
		},
	})
}

func TestDockerImageApplicationResource_InvalidPortsExposes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "nginx:latest"
					ports_exposes = "abc"
				`),
				ExpectError: regexp.MustCompile(`comma-separated list of port numbers`),
			},
		},
	})
}

func TestDockerImageApplicationResource_InvalidFQDN(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "nginx:latest"
					ports_exposes = "3000"
					fqdn          = "app.example.com"
				`),
				ExpectError: regexp.MustCompile(`must be a valid URL starting with http:// or https://`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testDockerImageResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_docker_image_application", "test", attrs)
}
