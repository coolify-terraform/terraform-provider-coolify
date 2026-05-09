package keydb_test

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

type mockKeydbState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	image       string
}

func newMockKeydbServer() (*httptest.Server, *mockKeydbState) {
	state := &mockKeydbState{
		uuid:  "keydb-test-uuid-001",
		name:  "keydb-test-db",
		image: "eqalpha/keydb:latest",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/keydb":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", state.uuid):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             state.uuid,
				"name":             state.name,
				"description":      state.description,
				"project_uuid":     "proj-uuid-1",
				"server_uuid":      "srv-uuid-1",
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

func TestKeydbDatabaseResource_Create(t *testing.T) {
	t.Parallel()
	srv, _ := newMockKeydbServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_keydb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "uuid", "keydb-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "name", "keydb-test-db"),
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "image", "eqalpha/keydb:latest"),
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "is_public", "false"),
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "environment_name", "production"),
				),
			},
		},
	})
}

func TestKeydbDatabaseResource_Update(t *testing.T) {
	t.Parallel()
	srv, _ := newMockKeydbServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_keydb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "name", "keydb-test-db"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_keydb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  name         = "updated-keydb"
  description  = "Updated KeyDB"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "name", "updated-keydb"),
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "description", "Updated KeyDB"),
				),
			},
		},
	})
}

func TestKeydbDatabaseResource_Import(t *testing.T) {
	t.Parallel()
	srv, _ := newMockKeydbServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_keydb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
			},
			{
				ResourceName:                         "coolify_keydb_database.test",
				ImportState:                          true,
				ImportStateId:                        "keydb-test-uuid-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
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
				"project_uuid": "proj-uuid-1", "server_uuid": "srv-uuid-1",
				"environment_name": "production", "image": "eqalpha/keydb:latest", "is_public": false,
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
resource "coolify_keydb_database" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_keydb_database.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_keydb_database.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/databases/"+rs.Primary.Attributes["uuid"], nil)
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
