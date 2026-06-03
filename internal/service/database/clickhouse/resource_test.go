package clickhouse_test

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

func TestClickhouseDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("clickhouse", "ch-test-db", "clickhouse/clickhouse-server:latest", map[string]interface{}{
		"clickhouse_admin_user":     "default",
		"clickhouse_admin_password": "secret123",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_clickhouse", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_clickhouse" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "name", "ch-test-db"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "image", "clickhouse/clickhouse-server:latest"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "clickhouse_admin_user", "default"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_log_drain_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_include_timestamps", "false"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "status", "running"),
				),
			},
			// Plan idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_clickhouse" "test" {
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
resource "coolify_database_clickhouse" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-ch"
  description  = "Updated ClickHouse"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "name", "updated-ch"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "description", "Updated ClickHouse"),
				),
			},
			// Update logging fields
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_clickhouse" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  name                  = "updated-ch"
  description           = "Updated ClickHouse"
  is_log_drain_enabled  = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_log_drain_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_include_timestamps", "true"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_database_clickhouse.test",
				ImportState:       true,
				ImportStateId:     "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"clickhouse_admin_password"},
			},
		},
	})
}

func TestClickhouseDatabaseResource_CreateWithTimestamps(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("clickhouse", "ch-ts-db", "clickhouse/clickhouse-server:latest", map[string]interface{}{
		"clickhouse_admin_user":     "default",
		"clickhouse_admin_password": "secret123",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_clickhouse" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  is_log_drain_enabled  = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_log_drain_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "is_include_timestamps", "true"),
				),
			},
		},
	})
}

func TestClickhouseDatabaseResource_CreateWithCredentials(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]interface{}
	mu := sync.Mutex{}
	deleted := false
	chUUID := "ch-creds-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/clickhouse":
			if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": chUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      chUUID,
				"name":                      "ch-creds-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "clickhouse/clickhouse-server:latest",
				"is_public":                 false,
				"clickhouse_admin_user":     "myadmin",
				"clickhouse_admin_password": "mypass123",
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
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
resource "coolify_database_clickhouse" "test" {
  project_uuid              = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid               = "bbbb0001-0001-4000-8000-000000000001"
  clickhouse_admin_user     = "myadmin"
  clickhouse_admin_password = "mypass123"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_clickhouse.test", "clickhouse_admin_user", "myadmin"),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if capturedBody == nil {
							return fmt.Errorf("Create request body was not captured")
						}
						if v, ok := capturedBody["clickhouse_admin_user"].(string); !ok || v != "myadmin" {
							return fmt.Errorf("expected clickhouse_admin_user=myadmin in Create body, got %v", capturedBody["clickhouse_admin_user"])
						}
						if v, ok := capturedBody["clickhouse_admin_password"].(string); !ok || v != "mypass123" {
							return fmt.Errorf("expected clickhouse_admin_password=mypass123 in Create body, got %v", capturedBody["clickhouse_admin_password"])
						}
						return nil
					},
				),
			},
		},
	})
}

func TestClickhouseDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const clickhouseUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/clickhouse":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": clickhouseUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", clickhouseUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      clickhouseUUID,
				"name":                      "ch-readback-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "clickhouse/clickhouse-server:latest",
				"is_public":                 false,
				"clickhouse_admin_user":     "default",
				"clickhouse_admin_password": "secret123",
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
			})

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", clickhouseUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", clickhouseUUID):
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
resource "coolify_database_clickhouse" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`(?s)ClickHouse database created but refresh failed.*Could not read ClickHouse database.*partial Terraform state was saved`),
		}},
	})
}

func TestClickhouseDatabaseResource_CreateAPIError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/clickhouse", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed: server not reachable"}`, http.StatusUnprocessableEntity)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_clickhouse" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`Error creating ClickHouse database`),
			},
		},
	})
}

func TestClickhouseDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	chUUID := "ch-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/clickhouse":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": chUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      chUUID,
				"name":                      "disappearing-ch",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "clickhouse/clickhouse-server:latest",
				"is_public":                 false,
				"clickhouse_admin_user":     "default",
				"clickhouse_admin_password": "secret",
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
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
resource "coolify_database_clickhouse" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_clickhouse.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_database_clickhouse.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
