package mysql_test

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

type mockMysqlState struct {
	mu            sync.Mutex
	uuid          string
	name          string
	description   string
	image         string
	mysqlUser     string
	mysqlPassword string
	mysqlDatabase string
	mysqlRootPwd  string
	deleted       bool
}

func newMockMysqlServer() (*httptest.Server, *mockMysqlState) {
	state := &mockMysqlState{
		uuid:          "mysql-test-uuid-001",
		name:          "mysql-test-db",
		image:         "mysql:8",
		mysqlUser:     "mysqluser",
		mysqlPassword: "mysqlpass",
		mysqlDatabase: "mydb",
		mysqlRootPwd:  "rootsecret",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mysql":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                state.uuid,
				"name":                state.name,
				"description":         state.description,
				"project_uuid":        "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":         "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":    "production",
				"image":               state.image,
				"is_public":           false,
				"public_port":         nil,
				"mysql_user":          state.mysqlUser,
				"mysql_password":      state.mysqlPassword,
				"mysql_database":      state.mysqlDatabase,
				"mysql_root_password": state.mysqlRootPwd,
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

func TestMysqlDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockMysqlServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_mysql_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mysql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "uuid", "mysql-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "name", "mysql-test-db"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "mysql_user", "mysqluser"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "mysql_database", "mydb"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "image", "mysql:8"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "is_public", "false"),
				),
			},
			// Plan idempotency
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mysql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
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

resource "coolify_mysql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-mysql-db"
  description  = "Updated MySQL"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "name", "updated-mysql-db"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "description", "Updated MySQL"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_mysql_database.test",
				ImportState:       true,
				ImportStateId:     "mysql-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"mysql_password", "mysql_root_password"},
			},
		},
	})
}

func TestMysqlDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	mysqlUUID := "mysql-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mysql":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": mysqlUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mysqlUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                mysqlUUID,
				"name":                "disappearing-mysql",
				"project_uuid":        "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":         "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":    "production",
				"image":               "mysql:8",
				"is_public":           false,
				"mysql_user":          "mysqluser",
				"mysql_password":      "secret",
				"mysql_database":      "mydb",
				"mysql_root_password": "rootsecret",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mysqlUUID):
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

resource "coolify_mysql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mysql_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_mysql_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
