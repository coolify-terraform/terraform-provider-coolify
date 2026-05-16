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

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
	deleted     bool
}

func newMockPostgresServer() (*httptest.Server, *mockPostgresState) {
	state := &mockPostgresState{
		uuid:       "aaaa0001-0001-4000-8000-000000000001",
		name:       "pg-test-db",
		image:      "postgres:16",
		pgUser:     "postgres",
		pgPassword: "secret123",
		pgDB:       "defaultdb",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/postgresql":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":              state.uuid,
				"name":              state.name,
				"description":       state.description,
				"project_uuid":      "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":       "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":  "production",
				"image":             state.image,
				"is_public":         false,
				"public_port":       nil,
				"postgres_user":               state.pgUser,
				"postgres_password":           state.pgPassword,
				"postgres_db":                 state.pgDB,
				"limits_memory":               "0",
				"limits_memory_swap":           "0",
				"limits_memory_swappiness":     60,
				"limits_memory_reservation":    "0",
				"limits_cpus":                  "0",
				"limits_cpuset":                "0",
				"limits_cpu_shares":            1024,
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
			state.deleted = true
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
	})))
	return srv, state
}

func TestPostgresqlDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockPostgresServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_postgresql_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_postgresql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "name", "pg-test-db"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "postgres_user", "postgres"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "postgres_db", "defaultdb"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "image", "postgres:16"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "is_public", "false"),
				),
			},
			// Plan idempotency: re-apply same config, expect empty plan
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_postgresql_database" "test" {
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
resource "coolify_postgresql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-pg-db"
  description  = "Updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "name", "updated-pg-db"),
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "description", "Updated description"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_postgresql_database.test",
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
	pgUUID := "pg-desc-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/postgresql":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": pgUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":              pgUUID,
				"name":              "pg-desc-db",
				"description":       description,
				"project_uuid":      "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":       "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":  "production",
				"image":             "postgres:16",
				"is_public":         false,
				"postgres_user":               "postgres",
				"postgres_password":           "secret",
				"postgres_db":                 "mydb",
				"limits_memory":               "0",
				"limits_memory_swap":           "0",
				"limits_memory_swappiness":     60,
				"limits_memory_reservation":    "0",
				"limits_cpus":                  "0",
				"limits_cpuset":                "0",
				"limits_cpu_shares":            1024,
			})

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", pgUUID):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["description"]; ok {
				if s, ok := v.(string); ok {
					description = s
				}
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete:
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
resource "coolify_postgresql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  description  = "initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_postgresql_database.test", "description", "initial"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					description = ""
					mu.Unlock()
				},
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_postgresql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("coolify_postgresql_database.test", "description"),
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
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":              pgUUID,
				"name":              "pg-readback-db",
				"project_uuid":      "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":       "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":  "production",
				"image":             "postgres:16",
				"is_public":         false,
				"postgres_user":               "postgres",
				"postgres_password":           "secret123",
				"postgres_db":                 "defaultdb",
				"limits_memory":               "0",
				"limits_memory_swap":           "0",
				"limits_memory_swappiness":     60,
				"limits_memory_reservation":    "0",
				"limits_cpus":                  "0",
				"limits_cpuset":                "0",
				"limits_cpu_shares":            1024,
			})

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
resource "coolify_postgresql_database" "test" {
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
resource "coolify_postgresql_database" "test" {
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
				"uuid":              pgUUID,
				"name":              "disappearing-pg",
				"project_uuid":      "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":       "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":  "production",
				"image":             "postgres:16",
				"is_public":         false,
				"postgres_user":     "postgres",
				"postgres_password":           "secret",
				"postgres_db":                 "mydb",
				"limits_memory":               "0",
				"limits_memory_swap":           "0",
				"limits_memory_swappiness":     60,
				"limits_memory_reservation":    "0",
				"limits_cpus":                  "0",
				"limits_cpuset":                "0",
				"limits_cpu_shares":            1024,
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
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_postgresql_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_postgresql_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_postgresql_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
