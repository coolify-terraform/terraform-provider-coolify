package dragonfly_test

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

type mockDragonflyState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
	deleted     bool
}

func newMockDragonflyServer() (*httptest.Server, *mockDragonflyState) {
	state := &mockDragonflyState{
		uuid:  "aaaa0001-0001-4000-8000-000000000001",
		name:  "dragonfly-test-db",
		image: "docker.dragonflydb.io/dragonflydb/dragonfly:latest",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/dragonfly":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      state.uuid,
				"name":                      state.name,
				"description":               state.description,
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     state.image,
				"is_public":                 false,
				"public_port":               nil,
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
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

func TestDragonflyDatabaseResource_Create(t *testing.T) {
	t.Parallel()
	srv, _ := newMockDragonflyServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_dragonfly_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "name", "dragonfly-test-db"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "image", "docker.dragonflydb.io/dragonflydb/dragonfly:latest"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "environment_name", "production"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
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

func TestDragonflyDatabaseResource_Update(t *testing.T) {
	t.Parallel()
	srv, _ := newMockDragonflyServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "name", "dragonfly-test-db"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  name         = "updated-dragonfly"
  description  = "Updated Dragonfly"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "name", "updated-dragonfly"),
					resource.TestCheckResourceAttr("coolify_dragonfly_database.test", "description", "Updated Dragonfly"),
				),
			},
		},
	})
}

func TestDragonflyDatabaseResource_Import(t *testing.T) {
	t.Parallel()
	srv, _ := newMockDragonflyServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_dragonfly_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			},
			{
				ResourceName:                         "coolify_dragonfly_database.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"dragonfly_password"},
			},
		},
	})
}

func TestDragonflyDatabaseResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	const dragonflyUUID = "aaaa0009-0009-4000-8000-000000000009"

	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/dragonfly":
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": dragonflyUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dragonflyUUID):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      dragonflyUUID,
				"name":                      "dragonfly-readback-db",
				"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":          "production",
				"image":                     "docker.dragonflydb.io/dragonflydb/dragonfly:latest",
				"is_public":                 false,
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
			})

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dragonflyUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dragonflyUUID):
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
resource "coolify_dragonfly_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`(?s)Dragonfly database created but refresh failed.*Could not read Dragonfly database.*partial Terraform state was saved`),
		}},
	})
}

func TestDragonflyDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	dbUUID := "dragonfly-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/dragonfly":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": dbUUID})
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dbUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid": dbUUID, "name": "disappearing-dragonfly",
				"project_uuid": "aaaa0001-0001-4000-8000-000000000001", "server_uuid": "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production", "image": "docker.dragonflydb.io/dragonflydb/dragonfly:latest", "is_public": false,
				"limits_memory":             "0",
				"limits_memory_swap":        "0",
				"limits_memory_swappiness":  60,
				"limits_memory_reservation": "0",
				"limits_cpus":               "0",
				"limits_cpuset":             "0",
				"limits_cpu_shares":         1024,
			})
		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", dbUUID):
			deleted = true
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/start"), strings.HasSuffix(r.URL.Path, "/stop"):
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
resource "coolify_dragonfly_database" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_dragonfly_database.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_dragonfly_database.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
