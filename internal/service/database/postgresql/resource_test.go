package postgresql_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

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
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
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

func TestPostgresqlDatabaseResource_Disappears(t *testing.T) {
	mu := sync.Mutex{}
	deleted := false
	pgUUID := "pg-disappear-uuid-001"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/postgresql":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": pgUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":              pgUUID,
				"name":              "disappearing-pg",
				"project_uuid":      "proj-uuid-1",
				"server_uuid":       "srv-uuid-1",
				"environment_name":  "production",
				"image":             "postgres:16",
				"is_public":         false,
				"postgres_user":     "postgres",
				"postgres_password": "secret",
				"postgres_db":       "mydb",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		case strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
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
					resource.TestCheckResourceAttrSet("coolify_postgresql_database.test", "uuid"),
					// Delete the database out-of-band via the mock API.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_postgresql_database.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/databases/"+uuid, nil)
						if err != nil {
							return err
						}
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
