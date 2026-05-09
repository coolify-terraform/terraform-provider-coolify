package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockServiceState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
}

func newMockServiceServer() (*httptest.Server, *mockServiceState) {
	state := &mockServiceState{
		uuid: "svc-test-uuid-001",
		name: "plausible-svc",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["name"].(string); ok && v != "" {
				state.name = v
			}
			if v, ok := body["description"].(string); ok {
				state.description = v
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":        state.uuid,
				"name":        state.name,
				"description": state.description,
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, state
}

func TestServiceResource_CreateImport(t *testing.T) {
	srv, _ := newMockServiceServer()
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

resource "coolify_service" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  type         = "plausible"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-test-uuid-001"),
					resource.TestCheckResourceAttr("coolify_service.test", "name", "plausible-svc"),
					resource.TestCheckResourceAttr("coolify_service.test", "type", "plausible"),
					resource.TestCheckResourceAttr("coolify_service.test", "environment_name", "production"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_service.test",
				ImportState:       true,
				ImportStateId:     "svc-test-uuid-001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"project_uuid", "server_uuid", "environment_name", "type"},
			},
		},
	})
}

func TestServiceResource_Disappears(t *testing.T) {
	mu := sync.Mutex{}
	deleted := false
	svcUUID := "svc-disappear-uuid-001"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": svcUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", svcUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid": svcUUID,
				"name": "disappearing-svc",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", svcUUID):
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
	}))
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

resource "coolify_service" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  type         = "plausible"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
					// Delete the service out-of-band via the mock API.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_service.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/services/"+uuid, nil)
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

func TestServiceResource_UpdateReturnsError(t *testing.T) {
	srv, _ := newMockServiceServer()
	defer srv.Close()

	baseConfig := func(desc string) string {
		return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_service" "test" {
  project_uuid = "proj-uuid-1"
  server_uuid  = "srv-uuid-1"
  type         = "plausible"
  description  = %q
}
`, srv.URL, desc)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: baseConfig("initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
				),
			},
			{
				Config:      baseConfig("updated description"),
				ExpectError: regexp.MustCompile(`Update not supported`),
			},
		},
	})
}
