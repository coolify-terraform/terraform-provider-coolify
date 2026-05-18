package mariadb_test

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

func TestMariadbDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("mariadb", "mariadb-test-db", "mariadb:11", map[string]interface{}{
		"mariadb_user":          "mariauser",
		"mariadb_password":      "mariapass",
		"mariadb_database":      "mariadb",
		"mariadb_root_password": "rootpwd",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_mariadb_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "name", "mariadb-test-db"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "mariadb_user", "mariauser"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "mariadb_database", "mariadb"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "image", "mariadb:11"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "is_log_drain_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "is_include_timestamps", "false"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "enable_ssl", "false"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "status", "running"),
				),
			},
			// Plan idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
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
resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-mariadb"
  description  = "Updated MariaDB"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "name", "updated-mariadb"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "description", "Updated MariaDB"),
				),
			},
			// Update SSL and log drain fields
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  name                  = "updated-mariadb"
  description           = "Updated MariaDB"
  enable_ssl            = true
  is_log_drain_enabled  = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "enable_ssl", "true"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "is_log_drain_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "is_include_timestamps", "true"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_mariadb_database.test",
				ImportState:       true,
				ImportStateId:     "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"mariadb_password", "mariadb_root_password"},
			},
		},
	})
}

func TestMariadbDatabaseResource_CreateWithSSLEnabled(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("mariadb", "mariadb-ssl-db", "mariadb:11", map[string]interface{}{
		"mariadb_user":          "mariadbuser",
		"mariadb_password":      "mariadbpass",
		"mariadb_database":      "mydb",
		"mariadb_root_password": "rootsecret",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  enable_ssl            = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "enable_ssl", "true"),
					resource.TestCheckResourceAttr("coolify_mariadb_database.test", "is_include_timestamps", "true"),
				),
			},
		},
	})
}

func TestMariadbDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const mariadbUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/mariadb":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": mariadbUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mariadbUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      mariadbUUID,
				"name":                      "mariadb-readback-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "mariadb:11",
				"is_public":                 false,
				"mariadb_user":              "mariauser",
				"mariadb_password":          "mariapass",
				"mariadb_database":          "mariadb",
				"mariadb_root_password":     "rootpwd",
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
			})

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mariadbUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", mariadbUUID):
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
resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`(?s)MariaDB database created but refresh failed.*Could not read MariaDB database.*partial Terraform state was saved`),
		}},
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
				"uuid":                      mdbUUID,
				"name":                      "disappearing-mariadb",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "mariadb:11",
				"is_public":                 false,
				"mariadb_user":              "mariauser",
				"mariadb_password":          "secret",
				"mariadb_database":          "mariadb",
				"mariadb_root_password":     "rootpwd",
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
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
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mariadb_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_mariadb_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestMariadbDatabaseResource_InvalidPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  public_port  = 99999
}
`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestMariadbDatabaseResource_InvalidUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_mariadb_database" "test" {
  project_uuid = "not-a-uuid"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: acctest.UUIDValidationError(),
			},
		},
	})
}
