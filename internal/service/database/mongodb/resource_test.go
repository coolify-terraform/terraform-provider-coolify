package mongodb_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
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
	deleted     bool
}

func newMockMongodbServer() (*httptest.Server, *mockMongodbState) {
	state := &mockMongodbState{
		uuid:      "aaaa0001-0001-4000-8000-000000000001",
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
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                       state.uuid,
				"name":                       state.name,
				"description":                state.description,
				"project_uuid":               "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":                "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":           "production",
				"image":                      state.image,
				"is_public":                  false,
				"public_port":                nil,
				"mongo_initdb_root_username": state.mongoUser,
				"mongo_initdb_root_password": state.mongoPass,
				"mongo_initdb_database":      state.mongoDB,
				"limits_memory":              "0",
				"limits_memory_swap":         "0",
				"limits_memory_swappiness":   60,
				"limits_memory_reservation":  "0",
				"limits_cpus":                "0",
				"limits_cpuset":              "0",
				"limits_cpu_shares":          1024,
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
			state.deleted = true
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_mongodb_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mongodb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "name", "mongo-test-db"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "mongo_initdb_root_username", "root"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "mongo_initdb_database", "admin"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "image", "mongo:7"),
				),
			},
			// Plan idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mongodb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mongodb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-mongo"
  description  = "Updated MongoDB"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "name", "updated-mongo"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "description", "Updated MongoDB"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_mongodb_database.test",
				ImportState:       true,
				ImportStateId:     "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"mongo_initdb_root_password"},
			},
		},
	})
}

func TestMongodbDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const mongoUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mongodb":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": mongoUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mongoUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                       mongoUUID,
				"name":                       "mongo-readback-db",
				"project_uuid":               "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":                "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":           "production",
				"image":                      "mongo:7",
				"is_public":                  false,
				"mongo_initdb_root_username": "root",
				"mongo_initdb_root_password": "mongosecret",
				"mongo_initdb_database":      "admin",
				"limits_memory":              "0",
				"limits_memory_swap":         "0",
				"limits_memory_swappiness":   60,
				"limits_memory_reservation":  "0",
				"limits_cpus":                "0",
				"limits_cpuset":              "0",
				"limits_cpu_shares":          1024,
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mongoUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mongodb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`(?s)MongoDB database created but refresh failed.*Could not read MongoDB database.*partial Terraform state was saved`),
		}},
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
				"project_uuid":               "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":                "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":           "production",
				"image":                      "mongo:7",
				"is_public":                  false,
				"mongo_initdb_root_username": "root",
				"mongo_initdb_root_password": "secret",
				"mongo_initdb_database":      "admin",
				"limits_memory":              "0",
				"limits_memory_swap":         "0",
				"limits_memory_swappiness":   60,
				"limits_memory_reservation":  "0",
				"limits_cpus":                "0",
				"limits_cpuset":              "0",
				"limits_cpu_shares":          1024,
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
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mongodb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mongodb_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_mongodb_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
