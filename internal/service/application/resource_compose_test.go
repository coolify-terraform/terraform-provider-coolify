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
// TestDockerComposeApplicationResource_Create
// ---------------------------------------------------------------------------

func TestDockerComposeApplicationResource_Create(t *testing.T) {
	app := client.Application{
		UUID:             "compose-app-uuid",
		Name:             "my-compose-app",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
		ProjectUUID:      "proj-uuid",
		ServerUUID:       "srv-uuid",
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
					name               = "my-compose-app"
					project_uuid       = "proj-uuid"
					server_uuid        = "srv-uuid"
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
					project_uuid       = "proj-uuid"
					server_uuid        = "srv-uuid"
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
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:             "compose-upd-uuid",
		Name:             "my-compose-app",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
		ProjectUUID:      "proj-uuid",
		ServerUUID:       "srv-uuid",
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
					project_uuid       = "proj-uuid"
					server_uuid        = "srv-uuid"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "docker_compose_raw", "version: '3'\nservices:\n  web:\n    image: nginx\n"),
				),
			},
			{
				Config: testDockerComposeResourceConfig(srv.URL, `
					name               = "my-compose-app"
					project_uuid       = "proj-uuid"
					server_uuid        = "srv-uuid"
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
	app := client.Application{
		UUID:             "compose-imp-uuid",
		Name:             "imported-compose-app",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx\n",
		ProjectUUID:      "proj-uuid",
		ServerUUID:       "srv-uuid",
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
					project_uuid       = "proj-uuid"
					server_uuid        = "srv-uuid"
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
			ProjectUUID:      "proj-uuid",
			ServerUUID:       "srv-uuid",
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
					project_uuid       = "proj-uuid"
					server_uuid        = "srv-uuid"
					docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_docker_compose_application.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_docker_compose_application.test"]
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

func testDockerComposeResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_docker_compose_application" "test" {
  %s
}
`, endpoint, attrs)
}
