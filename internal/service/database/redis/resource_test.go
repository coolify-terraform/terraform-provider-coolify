package redis_test

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

type mockRedisState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
	deleted     bool
}

func newMockRedisServer() (*httptest.Server, *mockRedisState) {
	state := &mockRedisState{
		uuid:  "redis-test-uuid-001",
		name:  "redis-test-db",
		image: "redis:7",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/redis":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             state.uuid,
				"name":             state.name,
				"description":      state.description,
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"image":            state.image,
				"is_public":        false,
				"public_port":      nil,
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

func TestRedisDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockRedisServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_redis_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_redis_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_redis_database.test", "uuid", "redis-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_redis_database.test", "name", "redis-test-db"),
					resource.TestCheckResourceAttr("coolify_redis_database.test", "image", "redis:7"),
					resource.TestCheckResourceAttr("coolify_redis_database.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_redis_database.test", "environment_name", "production"),
				),
			},
			// Plan idempotency
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_redis_database" "test" {
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

resource "coolify_redis_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-redis"
  description  = "Updated Redis"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_redis_database.test", "name", "updated-redis"),
					resource.TestCheckResourceAttr("coolify_redis_database.test", "description", "Updated Redis"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_redis_database.test",
				ImportState:       true,
				ImportStateId:     "redis-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
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
				"uuid":             redisUUID,
				"name":             "disappearing-redis",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"image":            "redis:7",
				"is_public":        false,
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", redisUUID):
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

resource "coolify_redis_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_redis_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_redis_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
