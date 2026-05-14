package service_test

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

const serviceTestConfig = `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"
}
`

func serviceConfig(serverURL string) string {
	return acctest.ProviderBlockForURL(serverURL) + serviceTestConfig
}

type mockServiceState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	deleted     bool
}

func newMockServiceServer() (*httptest.Server, *mockServiceState) {
	state := &mockServiceState{
		uuid: "dddd0001-0001-4000-8000-000000000001",
		name: "plausible-svc",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"type":             "plausible",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
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

func TestServiceResource_CreateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: serviceConfig(srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "dddd0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_service.test", "name", "plausible-svc"),
					resource.TestCheckResourceAttr("coolify_service.test", "type", "plausible"),
					resource.TestCheckResourceAttr("coolify_service.test", "environment_name", "production"),
				),
			},
			// Idempotency
			{
				Config:             serviceConfig(srv.URL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Import
			{
				ResourceName:      "coolify_service.test",
				ImportState:       true,
				ImportStateId:     "dddd0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"project_uuid", "server_uuid", "environment_name", "type"},
			},
		},
	})
}

func TestServiceResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	state := &mockServiceState{uuid: "dddd0009-0009-4000-8000-000000000009", name: "plausible-svc"}
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			forceReadFailure.Store(true)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             state.uuid,
				"name":             state.name,
				"description":      state.description,
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"type":             "plausible",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			state.deleted = true
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
				Config:      serviceConfig(srv.URL),
				ExpectError: regexp.MustCompile(`(?s)Service created but refresh failed.*Could not read service.*partial Terraform state was saved`),
			},
		},
	})
}

func TestServiceResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	svcUUID := "svc-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"uuid":             svcUUID,
				"name":             "disappearing-svc",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"type":             "plausible",
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
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_service.test", "/api/v1/services/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestServiceResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentDesc := "initial description"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": "svc-uuid-1"})

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["description"].(string); ok {
				currentDesc = v
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             "svc-uuid-1",
				"name":             "plausible-svc",
				"description":      currentDesc,
				"type":             "plausible",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
			})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/services/svc-uuid-"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             "svc-uuid-1",
				"name":             "plausible-svc",
				"description":      currentDesc,
				"type":             "plausible",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
			})

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	baseConfig := func(desc string) string {
		return acctest.ProviderBlockForURL(srv.URL) + fmt.Sprintf(`
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"
  description  = %q
}
`, desc)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: baseConfig("initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-uuid-1"),
					resource.TestCheckResourceAttr("coolify_service.test", "description", "initial description"),
				),
			},
			{
				Config: baseConfig("updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// UUID stays the same, proving in-place update (no destroy+recreate).
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-uuid-1"),
					resource.TestCheckResourceAttr("coolify_service.test", "description", "updated description"),
				),
			},
		},
	})
}
