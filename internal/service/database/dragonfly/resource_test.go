package dragonfly_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type mockDragonflyState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
}

func newMockDragonflyServer() (*httptest.Server, *mockDragonflyState) {
	state := &mockDragonflyState{
		uuid:  "dragonfly-test-uuid-001",
		name:  "dragonfly-test-db",
		image: "docker.dragonflydb.io/dragonflydb/dragonfly:latest",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/dragonfly":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             state.uuid,
				"name":             state.name,
				"description":      state.description,
				"project_uuid":     "proj-uuid-1",
				"server_uuid":      "srv-uuid-1",
				"environment_name": "production",
				"image":            state.image,
				"is_public":        false,
				"public_port":      nil,
			})

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["name"].(string); ok {
				state.name = v
			}
			if v, ok := body["description"].(string); ok {
				state.description = v
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, state
}

func TestDragonflyDatabaseResource_Create(t *testing.T) {
	srv, _ := newMockDragonflyServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "uuid", "dragonfly-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "name", "dragonfly-test-db"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "image", "docker.dragonflydb.io/dragonflydb/dragonfly:latest"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "environment_name", "production"),
				),
			},
		},
	})
}

func TestDragonflyDatabaseResource_Update(t *testing.T) {
	srv, _ := newMockDragonflyServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "name", "dragonfly-test-db"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  name         = "updated-dragonfly"
  description  = "Updated Dragonfly"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "name", "updated-dragonfly"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "description", "Updated Dragonfly"),
				),
			},
		},
	})
}

func TestDragonflyDatabaseResource_Import(t *testing.T) {
	srv, _ := newMockDragonflyServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
			},
			{
				ResourceName:                         "coolify_dragonfly_database.test",
				ImportState:                          true,
				ImportStateId:                        "dragonfly-test-uuid-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}
