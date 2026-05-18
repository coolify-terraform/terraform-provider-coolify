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
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/dbtest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMongodbDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("mongodb", "mongo-test-db", "mongo:7", map[string]interface{}{
		"mongo_initdb_root_username": "root",
		"mongo_initdb_root_password": "mongosecret",
		"mongo_initdb_database":      "admin",
	})
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
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "is_log_drain_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "is_include_timestamps", "false"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "enable_ssl", "false"),
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
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
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

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mongoUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

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

func TestMongodbDatabaseResource_InvalidSSLMode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mongodb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  ssl_mode     = "bogus"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}
