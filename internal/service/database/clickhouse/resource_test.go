package clickhouse_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockClickhouseState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
	adminUser   string
	adminPass   string
}

func newMockClickhouseServer() (*httptest.Server, *mockClickhouseState) {
	state := &mockClickhouseState{
		uuid:      "ch-test-uuid-001",
		name:      "ch-test-db",
		image:     "clickhouse/clickhouse-server:latest",
		adminUser: "default",
		adminPass: "secret123",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/clickhouse":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      state.uuid,
				"name":                      state.name,
				"description":               state.description,
				"project_uuid":              "proj-uuid-1",
				"server_uuid":               "srv-uuid-1",
				"environment_name":          "production",
				"image":                     state.image,
				"is_public":                 false,
				"public_port":               nil,
				"clickhouse_admin_user":     state.adminUser,
				"clickhouse_admin_password": state.adminPass,
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

func TestClickhouseDatabaseResource_CreateUpdateImport(t *testing.T) {
	srv, _ := newMockClickhouseServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_clickhouse_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "uuid", "ch-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "name", "ch-test-db"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "image", "clickhouse/clickhouse-server:latest"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "clickhouse_admin_user", "default"),
				),
			},
			// Plan idempotency
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_clickhouse_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
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

resource "coolify_clickhouse_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  name         = "updated-ch"
  description  = "Updated ClickHouse"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "name", "updated-ch"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "description", "Updated ClickHouse"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_clickhouse_database.test",
				ImportState:       true,
				ImportStateId:     "ch-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"clickhouse_admin_password"},
			},
		},
	})
}

func TestClickhouseDatabaseResource_CreateWithCredentials(t *testing.T) {
	var capturedBody map[string]interface{}
	mu := sync.Mutex{}
	chUUID := "ch-creds-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/clickhouse":
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": chUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":                      chUUID,
				"name":                      "ch-creds-db",
				"project_uuid":              "proj-uuid-1",
				"server_uuid":               "srv-uuid-1",
				"environment_name":          "production",
				"image":                     "clickhouse/clickhouse-server:latest",
				"is_public":                 false,
				"clickhouse_admin_user":     "myadmin",
				"clickhouse_admin_password": "mypass123",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
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
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_clickhouse_database" "test" {
  project_uuid              = "proj-uuid-1"
  server_uuid               = "srv-uuid-1"
  clickhouse_admin_user     = "myadmin"
  clickhouse_admin_password = "mypass123"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "clickhouse_admin_user", "myadmin"),
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

func TestClickhouseDatabaseResource_Disappears(t *testing.T) {
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
				"project_uuid":              "proj-uuid-1",
				"server_uuid":               "srv-uuid-1",
				"environment_name":          "production",
				"image":                     "clickhouse/clickhouse-server:latest",
				"is_public":                 false,
				"clickhouse_admin_user":     "default",
				"clickhouse_admin_password": "secret",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", chUUID):
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

resource "coolify_clickhouse_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_clickhouse_database.test", "uuid"),
					// Delete the database out-of-band via the mock API.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_clickhouse_database.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/databases/"+uuid, nil)
						if err != nil {
							return err
						}
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
