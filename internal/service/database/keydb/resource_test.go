package keydb_test

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

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/dbtest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestKeydbDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("keydb", "keydb-test-db", "eqalpha/keydb:latest", nil)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_keydb", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "name", "keydb-test-db"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "image", "eqalpha/keydb:latest"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "status", "running"),
				),
			},
			// Plan idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
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
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-keydb"
  description  = "Updated KeyDB"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "name", "updated-keydb"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "description", "Updated KeyDB"),
				),
			},
			// Update limits (Coolify PATCH allow-list field)
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  name                  = "updated-keydb"
  description           = "Updated KeyDB"
  limits_memory         = "512M"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "limits_memory", "512M"),
				),
			},
			// Update health check fields to non-default values
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
  name                    = "updated-keydb"
  description             = "Updated KeyDB"
  health_check_enabled    = false
  health_check_interval   = 30
  health_check_timeout    = 10
  health_check_retries    = 3
  health_check_start_period = 15
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "health_check_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "health_check_interval", "30"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "health_check_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "health_check_retries", "3"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "health_check_start_period", "15"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_database_keydb.test",
				ImportState:       true,
				ImportStateId:     "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"keydb_password"},
			},
		},
	})
}

func TestKeydbDatabaseResource_Create(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("keydb", "keydb-test-db", "eqalpha/keydb:latest", nil)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_keydb", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "name", "keydb-test-db"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "image", "eqalpha/keydb:latest"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "status", "running"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestKeydbDatabaseResource_Update(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("keydb", "keydb-test-db", "eqalpha/keydb:latest", nil)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "name", "keydb-test-db"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-keydb"
  description  = "Updated KeyDB"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "name", "updated-keydb"),
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "description", "Updated KeyDB"),
				),
			},
			// Update limits (Coolify PATCH allow-list field)
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  name                  = "updated-keydb"
  description           = "Updated KeyDB"
  limits_memory         = "512M"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "limits_memory", "512M"),
				),
			},
		},
	})
}

func TestKeydbDatabaseResource_CreateWithLimitsMemory(t *testing.T) {
	t.Parallel()
	srv, state := dbtest.NewMockServer("keydb", "keydb-ssl-db", "eqalpha/keydb:latest", nil)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid  = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid   = "bbbb0001-0001-4000-8000-000000000001"
  limits_memory = "512M"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_keydb.test", "limits_memory", "512M"),
					func(_ *terraform.State) error {
						if v, ok := state.LastCreate["limits_memory"]; !ok || v != "512M" {
							return fmt.Errorf("create POST limits_memory = %v, want 512M", state.LastCreate["limits_memory"])
						}
						for _, k := range []string{"is_log_drain_enabled", "is_include_timestamps", "enable_ssl", "ssl_mode"} {
							if _, ok := state.LastCreate[k]; ok {
								return fmt.Errorf("create POST sent unallowed key %s", k)
							}
							if _, ok := state.LastPatch[k]; ok {
								return fmt.Errorf("create PATCH sent unallowed key %s", k)
							}
						}
						return nil
					},
				),
			},
		},
	})
}

func TestKeydbDatabaseResource_Import(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("keydb", "keydb-test-db", "eqalpha/keydb:latest", nil)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:                         "coolify_database_keydb.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"keydb_password"},
			},
		},
	})
}

func TestKeydbDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const keydbUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/keydb":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": keydbUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", keydbUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      keydbUUID,
				"name":                      "keydb-readback-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "eqalpha/keydb:latest",
				"is_public":                 false,
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
				"health_check_enabled":      true,
				"health_check_interval":     15,
				"health_check_timeout":      5,
				"health_check_retries":      5,
				"health_check_start_period": 5,
			})

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", keydbUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", keydbUUID):
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
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`KeyDB database created but refresh failed`),
		}},
	})
}

func TestKeydbDatabaseResource_CreateAPIError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/keydb", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed: server not reachable"}`, http.StatusUnprocessableEntity)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`Error creating KeyDB database`),
			},
		},
	})
}

func TestKeydbDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	dbUUID := "keydb-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/keydb":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": dbUUID})
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dbUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid": dbUUID, "name": "disappearing-keydb",
				"project_uuid": "aaaa0001-0001-4000-8000-000000000001", "server_uuid": "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production", "image": "eqalpha/keydb:latest", "is_public": false,
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
				"health_check_enabled":      true,
				"health_check_interval":     15,
				"health_check_timeout":      5,
				"health_check_retries":      5,
				"health_check_start_period": 5,
			})
		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dbUUID):
			deleted = true
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"),
			r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/stop"):
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
resource "coolify_database_keydb" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_keydb.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_database_keydb.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
