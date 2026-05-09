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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Create
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Create(t *testing.T) {
	app := client.Application{
		UUID:                    "docker-app-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "proj-uuid",
		ServerUUID:              "srv-uuid",
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

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
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
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
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
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:                    "docker-upd-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "proj-uuid",
		ServerUUID:              "srv-uuid",
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

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
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
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
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
	app := client.Application{
		UUID:                    "docker-imp-uuid",
		Name:                    "imported-docker-app",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "proj-uuid",
		ServerUUID:              "srv-uuid",
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

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "imported-docker-app"
					project_uuid   = "proj-uuid"
					server_uuid    = "srv-uuid"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
			},
			{
				ResourceName:                         "coolify_docker_image_application.test",
				ImportState:                          true,
				ImportStateId:                        "docker-imp-uuid",
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
			ProjectUUID:             "proj-uuid",
			ServerUUID:              "srv-uuid",
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid  = "proj-uuid"
					server_uuid   = "srv-uuid"
					docker_image  = "nginx:latest"
					ports_exposes = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_docker_image_application.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_docker_image_application.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/applications/"+rs.Primary.Attributes["uuid"], nil)
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testDockerImageResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_docker_image_application" "test" {
  %s
}
`, endpoint, attrs)
}
