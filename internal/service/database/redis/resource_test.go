package redis_test

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
)

func TestRedisDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("redis", "redis-test-db", "redis:7", map[string]interface{}{
		"redis_password": "default-redis-pass",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_redis", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_redis" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_redis.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "name", "redis-test-db"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "image", "redis:7"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "is_log_drain_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "is_include_timestamps", "false"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "enable_ssl", "false"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "status", "running"),
				),
			},
			// Plan idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_redis" "test" {
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
resource "coolify_database_redis" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-redis"
  description  = "Updated Redis"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_redis.test", "name", "updated-redis"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "description", "Updated Redis"),
				),
			},
			// Update SSL and log drain fields
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_redis" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  name                  = "updated-redis"
  description           = "Updated Redis"
  enable_ssl            = true
  is_log_drain_enabled  = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_redis.test", "enable_ssl", "true"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "is_log_drain_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "is_include_timestamps", "true"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_database_redis.test",
				ImportState:       true,
				ImportStateId:     "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"redis_password"},
			},
		},
	})
}

func TestRedisDatabaseResource_CreateWithSSLEnabled(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServer("redis", "redis-ssl-db", "redis:7", map[string]interface{}{
		"redis_password": "redis-ssl-pass",
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_redis" "test" {
  project_uuid          = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
  enable_ssl            = true
  is_include_timestamps = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_redis.test", "enable_ssl", "true"),
					resource.TestCheckResourceAttr("coolify_database_redis.test", "is_include_timestamps", "true"),
				),
			},
		},
	})
}

func TestRedisDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const redisUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/redis":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": redisUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", redisUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      redisUUID,
				"name":                      "redis-readback-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "redis:7",
				"is_public":                 false,
				"redis_password":            "default-redis-pass",
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

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", redisUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", redisUUID):
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
resource "coolify_database_redis" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`(?s)Redis database created but refresh failed.*Could not read Redis database.*partial Terraform state was saved`),
		}},
	})
}

func TestRedisDatabaseResource_CreateAPIError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/redis", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed: server not reachable"}`, http.StatusUnprocessableEntity)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_redis" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`Error creating Redis database`),
			},
		},
	})
}

func TestRedisDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	redisUUID := "redis-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/redis":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": redisUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", redisUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      redisUUID,
				"name":                      "disappearing-redis",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "redis:7",
				"is_public":                 false,
				"redis_password":            "default-redis-pass",
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

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", redisUUID):
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
resource "coolify_database_redis" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_redis.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_database_redis.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
