package mongodb_test

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

type mockMongodbState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
	mongoUser   string
	mongoPass   string
	mongoDB     string
}

func newMockMongodbServer() (*httptest.Server, *mockMongodbState) {
	state := &mockMongodbState{
		uuid:      "mongo-test-uuid-001",
		name:      "mongo-test-db",
		image:     "mongo:7",
		mongoUser: "root",
		mongoPass: "mongosecret",
		mongoDB:   "admin",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mongodb":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                       state.uuid,
				"name":                       state.name,
				"description":                state.description,
				"project_uuid":               "proj-uuid-1",
				"server_uuid":                "srv-uuid-1",
				"environment_name":           "production",
				"image":                      state.image,
				"is_public":                  false,
				"public_port":                nil,
				"mongo_initdb_root_username": state.mongoUser,
				"mongo_initdb_root_password": state.mongoPass,
				"mongo_initdb_database":      state.mongoDB,
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
	})))
	return srv, state
}

func TestMongodbDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockMongodbServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mongodb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "uuid", "mongo-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "name", "mongo-test-db"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "mongo_initdb_root_username", "root"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "mongo_initdb_database", "admin"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "image", "mongo:7"),
				),
			},
			// Plan idempotency
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mongodb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`, srv.URL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mongodb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  name         = "updated-mongo"
  description  = "Updated MongoDB"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "name", "updated-mongo"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "description", "Updated MongoDB"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_mongodb_database.test",
				ImportState:       true,
				ImportStateId:     "mongo-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"mongo_initdb_root_password"},
			},
		},
	})
}

func TestMongodbDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	mongoUUID := "mongo-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mongodb":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": mongoUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mongoUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                       mongoUUID,
				"name":                       "disappearing-mongo",
				"project_uuid":               "proj-uuid-1",
				"server_uuid":                "srv-uuid-1",
				"environment_name":           "production",
				"image":                      "mongo:7",
				"is_public":                  false,
				"mongo_initdb_root_username": "root",
				"mongo_initdb_root_password": "secret",
				"mongo_initdb_database":      "admin",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mongoUUID):
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
	})))
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

resource "coolify_mongodb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mongodb_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_mongodb_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
