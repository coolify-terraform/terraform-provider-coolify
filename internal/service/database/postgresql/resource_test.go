package postgresql_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

type mockPostgresState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
	pgUser      string
	pgPassword  string
	pgDB        string
}

func newMockPostgresServer() (*httptest.Server, *mockPostgresState) {
	state := &mockPostgresState{
		uuid:       "pg-test-uuid-001",
		name:       "pg-test-db",
		image:      "postgres:16",
		pgUser:     "postgres",
		pgPassword: "secret123",
		pgDB:       "defaultdb",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/postgresql":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":              state.uuid,
				"name":              state.name,
				"description":       state.description,
				"project_uuid":      "proj-uuid-1",
				"server_uuid":       "srv-uuid-1",
				"environment_name":  "production",
				"image":             state.image,
				"is_public":         false,
				"public_port":       nil,
				"postgres_user":     state.pgUser,
				"postgres_password": state.pgPassword,
				"postgres_db":       state.pgDB,
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
			if v, ok := body["image"].(string); ok {
				state.image = v
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "started"})

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "stopped"})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
	return srv, state
}

func TestPostgresqlDatabaseResource_CreateUpdateImport(t *testing.T) {
	srv, state := newMockPostgresServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_postgresql_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "uuid", "pg-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "name", "pg-test-db"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "postgres_user", "postgres"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "postgres_db", "defaultdb"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "image", "postgres:16"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "is_public", "false"),
				),
			},
			// Update name and description
			{
				PreConfig: func() {
					state.mu.Lock()
					defer state.mu.Unlock()
					// The mock PATCH handler will update these
				},
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_postgresql_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  name         = "updated-pg-db"
  description  = "Updated description"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "uuid", "pg-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "name", "updated-pg-db"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "description", "Updated description"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_postgresql_database.test",
				ImportState:       true,
				ImportStateId:     "pg-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"postgres_password"},
			},
		},
	})
}
