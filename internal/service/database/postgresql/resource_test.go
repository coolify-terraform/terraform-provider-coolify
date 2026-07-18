package postgresql_test

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

func TestPostgresqlDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_postgresql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "name", "pg-test-db"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "postgres_user", "postgres"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "postgres_db", "defaultdb"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "image", "postgres:16"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "is_log_drain_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "is_include_timestamps", "false"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "enable_ssl", "false"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "status", "running"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "limits_cpu_shares", "1024"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "instant_deploy", "false"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_interval", "15"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_timeout", "5"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_retries", "5"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_start_period", "5"),
				),
			},
			// Plan idempotency: re-apply same config, expect empty plan
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update name and description
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-pg-db"
  description  = "Updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "name", "updated-pg-db"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "description", "Updated description"),
				),
			},
			// Update SSL and log drain fields
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  name                  = "updated-pg-db"
  description           = "Updated description"
  enable_ssl            = true
  ssl_mode              = "require"
  is_log_drain_enabled  = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "enable_ssl", "true"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "ssl_mode", "require"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "is_log_drain_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "is_include_timestamps", "true"),
				),
			},
			// Update health check fields to non-default values
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
  name                    = "updated-pg-db"
  description             = "Updated description"
  enable_ssl              = true
  ssl_mode                = "require"
  is_log_drain_enabled    = true
  is_include_timestamps   = true
  health_check_enabled    = false
  health_check_interval   = 30
  health_check_timeout    = 10
  health_check_retries    = 3
  health_check_start_period = 15
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_interval", "30"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_retries", "3"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "health_check_start_period", "15"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_database_postgresql.test",
				ImportState:       true,
				ImportStateId:     "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"postgres_password"},
			},
		},
	})
}

func TestPostgresqlDatabaseResource_DescriptionNullHandling(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	description := "initial"
	deleted := false
	pgUUID := "pg-desc-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/postgresql":
			deleted = false
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": pgUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      pgUUID,
				"name":                      "pg-desc-db",
				"description":               description,
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "postgres:16",
				"is_public":                 false,
				"postgres_user":             "postgres",
				"postgres_password":         "secret",
				"postgres_db":               "mydb",
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

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			if v, ok := body["description"]; ok {
				if s, ok := v.(string); ok {
					description = s
				}
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

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
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  description  = "initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "description", "initial"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					description = ""
					deleted = false
					mu.Unlock()
				},
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("coolify_database_postgresql.test", "description"),
				),
			},
		},
	})
}

func TestPostgresqlDatabaseResource_InternalDBUrlAndInstantDeploy(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-url-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "mydb",
		"internal_db_url":   "postgresql://postgres:secret123@pg-url-db:5432/mydb",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid   = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid    = "bbbb0001-0001-4000-8000-000000000001"
  instant_deploy = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "internal_db_url", "postgresql://postgres:secret123@pg-url-db:5432/mydb"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "instant_deploy", "true"),
				),
			},
		},
	})
}

func TestPostgresqlDatabaseResource_CreateWithSSLEnabled(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-ssl-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "ssldb",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  enable_ssl            = true
  ssl_mode              = "require"
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "enable_ssl", "true"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "ssl_mode", "require"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "is_include_timestamps", "true"),
				),
			},
		},
	})
}

func TestPostgresqlDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const pgUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/postgresql":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": pgUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      pgUUID,
				"name":                      "pg-readback-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "postgres:16",
				"is_public":                 false,
				"postgres_user":             "postgres",
				"postgres_password":         "secret123",
				"postgres_db":               "defaultdb",
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

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
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
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`(?s)PostgreSQL database created but refresh failed.*Could not read PostgreSQL database.*partial Terraform state was saved`),
		}},
	})
}

func TestPostgresqlDatabaseResource_InvalidPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
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

func TestPostgresqlDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	pgUUID := "pg-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"uuid":                      pgUUID,
				"name":                      "disappearing-pg",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "postgres:16",
				"is_public":                 false,
				"postgres_user":             "postgres",
				"postgres_password":         "secret",
				"postgres_db":               "mydb",
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

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			deleted = true
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
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_postgresql.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_database_postgresql.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPostgresqlDatabaseResource_ImportCompound
// ---------------------------------------------------------------------------

func TestPostgresqlDatabaseResource_ImportCompound(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	})
	defer srv.Close()

	const (
		projUUID = "aaaa0001-0001-4000-8000-000000000001"
		srvUUID  = "bbbb0001-0001-4000-8000-000000000001"
		dbUUID   = "aaaa0001-0001-4000-8000-000000000001"
		envName  = "production"
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:  "coolify_database_postgresql.test",
				ImportState:   true,
				ImportStateId: projUUID + ":" + srvUUID + ":" + envName + ":" + dbUUID,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(states))
					}
					attrs := states[0].Attributes
					checks := map[string]string{
						"project_uuid":     projUUID,
						"server_uuid":      srvUUID,
						"environment_name": envName,
						"uuid":             dbUUID,
					}
					for k, want := range checks {
						if got := attrs[k]; got != want {
							return fmt.Errorf("attribute %s = %q, want %q", k, got, want)
						}
					}
					return nil
				},
			},
		},
	})
}

func TestPostgresqlDatabaseResource_ImportBadSimpleUUID(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:  "coolify_database_postgresql.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestPostgresqlDatabaseResource_ImportCompoundWrongServer(t *testing.T) {
	t.Parallel()
	srv, state := dbtest.NewMockServer("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	})
	defer srv.Close()

	const (
		projUUID     = "aaaa0001-0001-4000-8000-000000000001"
		wrongSrvUUID = "bbbb0002-0002-4000-8000-000000000002"
		envName      = "production"
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:  "coolify_database_postgresql.test",
				ImportState:   true,
				ImportStateId: projUUID + ":" + wrongSrvUUID + ":" + envName + ":" + state.UUID,
				ExpectError:   regexp.MustCompile(`is not deployed on server`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPostgresqlDatabaseResource_ImportCompoundBadParts
// ---------------------------------------------------------------------------

func TestPostgresqlDatabaseResource_ImportCompoundBadParts(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:  "coolify_database_postgresql.test",
				ImportState:   true,
				ImportStateId: "a:b:c",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestPostgresqlDatabaseResource_ImportCompoundEmptyEnv
// ---------------------------------------------------------------------------

func TestPostgresqlDatabaseResource_ImportCompoundEmptyEnv(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:  "coolify_database_postgresql.test",
				ImportState:   true,
				ImportStateId: "aaaa0001-0001-4000-8000-000000000001:bbbb0001-0001-4000-8000-000000000001::aaaa0001-0001-4000-8000-000000000001",
				ExpectError:   regexp.MustCompile(`environment_name must not be empty`),
			},
		},
	})
}

func TestPostgresqlDatabaseResource_CreateAPIError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/postgresql", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed: server not reachable"}`, http.StatusUnprocessableEntity)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`Error creating PostgreSQL database`),
			},
		},
	})
}

func TestPostgresqlDatabaseResource_InvalidSSLMode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
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
