package mariadb_test

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

type mockMariadbState struct {
	mu              sync.Mutex
	uuid            string
	name            string
	description     string
	image           string
	mariadbUser     string
	mariadbPassword string
	mariadbDatabase string
	mariadbRootPwd  string
	deleted         bool
}

func newMockMariadbServer() (*httptest.Server, *mockMariadbState) {
	state := &mockMariadbState{
		uuid:            "mdb-test-uuid-001",
		name:            "mariadb-test-db",
		image:           "mariadb:11",
		mariadbUser:     "mariauser",
		mariadbPassword: "mariapass",
		mariadbDatabase: "mariadb",
		mariadbRootPwd:  "rootpwd",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mariadb":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                  state.uuid,
				"name":                  state.name,
				"description":           state.description,
				"project_uuid":          "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":           "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":      "production",
				"image":                 state.image,
				"is_public":             false,
				"public_port":           nil,
				"mariadb_user":          state.mariadbUser,
				"mariadb_password":      state.mariadbPassword,
				"mariadb_database":      state.mariadbDatabase,
				"mariadb_root_password": state.mariadbRootPwd,
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

func TestMariadbDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockMariadbServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_mariadb_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "uuid", "mdb-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "name", "mariadb-test-db"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "mariadb_user", "mariauser"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "mariadb_database", "mariadb"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "image", "mariadb:11"),
				),
			},
			// Plan idempotency
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_mariadb_database" "test" {
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

resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-mariadb"
  description  = "Updated MariaDB"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "name", "updated-mariadb"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "description", "Updated MariaDB"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_mariadb_database.test",
				ImportState:       true,
				ImportStateId:     "mdb-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"mariadb_password", "mariadb_root_password"},
			},
		},
	})
}

func TestMariadbDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	mdbUUID := "mdb-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mariadb":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": mdbUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mdbUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                  mdbUUID,
				"name":                  "disappearing-mariadb",
				"project_uuid":          "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":           "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":      "production",
				"image":                 "mariadb:11",
				"is_public":             false,
				"mariadb_user":          "mariauser",
				"mariadb_password":      "secret",
				"mariadb_database":      "mariadb",
				"mariadb_root_password": "rootpwd",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mdbUUID):
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

resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mariadb_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_mariadb_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
