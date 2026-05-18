package environment_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// mockEnvironment stores environment data in the mock server.
type mockEnvironment struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ProjectUUID string `json:"project_uuid"`
	Description string `json:"description,omitempty"`
}

// mockEnvironmentStore is a thread-safe in-memory store for mock environments.
type mockEnvironmentStore struct {
	mu      sync.RWMutex
	envs    map[string]*mockEnvironment // composite key "projectUUID:name"
	counter int64
}

func newMockEnvironmentStore() *mockEnvironmentStore {
	return &mockEnvironmentStore{
		envs: make(map[string]*mockEnvironment),
	}
}

func (s *mockEnvironmentStore) key(projectUUID, name string) string {
	return projectUUID + ":" + name
}

func (s *mockEnvironmentStore) Create(projectUUID, name, description string) *mockEnvironment {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	env := &mockEnvironment{
		ID:          s.counter,
		Name:        name,
		ProjectUUID: projectUUID,
		Description: description,
	}
	s.envs[s.key(projectUUID, name)] = env
	return env
}

func (s *mockEnvironmentStore) Get(projectUUID, name string) (*mockEnvironment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.envs[s.key(projectUUID, name)]
	return env, ok
}

func (s *mockEnvironmentStore) Delete(projectUUID, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(projectUUID, name)
	_, ok := s.envs[k]
	if ok {
		delete(s.envs, k)
	}
	return ok
}

func (s *mockEnvironmentStore) List(projectUUID string) []*mockEnvironment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*mockEnvironment, 0)
	for _, env := range s.envs {
		if env.ProjectUUID == projectUUID {
			result = append(result, env)
		}
	}
	return result
}

// newMockEnvironmentServer creates an httptest.Server that simulates the Coolify API for environments.
func newMockEnvironmentServer(auditT ...testing.TB) (*httptest.Server, *mockEnvironmentStore) {
	store := newMockEnvironmentStore()
	mux := http.NewServeMux()

	// POST /api/v1/projects/{projectUUID}/environments
	mux.HandleFunc("POST /api/v1/projects/{projectUUID}/environments", func(w http.ResponseWriter, r *http.Request) {
		projectUUID := r.PathValue("projectUUID")
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		env := store.Create(projectUUID, body.Name, body.Description)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(env)
	})

	// GET /api/v1/projects/{projectUUID}/environments
	mux.HandleFunc("GET /api/v1/projects/{projectUUID}/environments", func(w http.ResponseWriter, r *http.Request) {
		projectUUID := r.PathValue("projectUUID")
		envs := store.List(projectUUID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envs)
	})

	// GET /api/v1/projects/{projectUUID}/{envName}
	mux.HandleFunc("GET /api/v1/projects/{projectUUID}/{envName}", func(w http.ResponseWriter, r *http.Request) {
		projectUUID := r.PathValue("projectUUID")
		envName := r.PathValue("envName")
		env, ok := store.Get(projectUUID, envName)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(env)
	})

	// DELETE /api/v1/projects/{projectUUID}/environments/{envName}
	mux.HandleFunc("DELETE /api/v1/projects/{projectUUID}/environments/{envName}", func(w http.ResponseWriter, r *http.Request) {
		projectUUID := r.PathValue("projectUUID")
		envName := r.PathValue("envName")
		if !store.Delete(projectUUID, envName) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	handler := acctest.WithVersionEndpoint(mux)
	if len(auditT) > 0 {
		handler = spectest.WithSpecAudit(auditT[0], "coolify-v4", handler)
	}
	server := httptest.NewServer(handler)
	return server, store
}

// checkEnvironmentDestroy verifies environments are deleted after test completion.
func checkEnvironmentDestroy(serverURL string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "coolify_environment" {
				continue
			}
			projectUUID := rs.Primary.Attributes["project_uuid"]
			name := rs.Primary.Attributes["name"]
			if projectUUID == "" || name == "" {
				continue
			}
			path := fmt.Sprintf("/api/v1/projects/%s/%s", url.PathEscape(projectUUID), url.PathEscape(name))
			req, err := http.NewRequest(http.MethodGet, serverURL+path, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("error checking destroy for coolify_environment %s/%s: %w", projectUUID, name, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("coolify_environment %s/%s still exists (status %d)", projectUUID, name, resp.StatusCode)
			}
		}
		return nil
	}
}

func TestEnvironmentResource_Create(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer(t)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkEnvironmentDestroy(server.URL),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "staging"
  description  = "Staging environment"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment.test", "id"),
					resource.TestCheckResourceAttr("coolify_environment.test", "project_uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_environment.test", "name", "staging"),
					resource.TestCheckResourceAttr("coolify_environment.test", "description", "Staging environment"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "staging"
  description  = "Staging environment"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestEnvironmentResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	store := newMockEnvironmentStore()
	var forceReadFailure atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/projects/{projectUUID}/environments", func(w http.ResponseWriter, r *http.Request) {
		projectUUID := r.PathValue("projectUUID")
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		env := store.Create(projectUUID, body.Name, "")
		forceReadFailure.Store(true)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(env)
	})
	mux.HandleFunc("GET /api/v1/projects/{projectUUID}/{envName}", func(w http.ResponseWriter, r *http.Request) {
		if forceReadFailure.Load() {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		projectUUID := r.PathValue("projectUUID")
		envName := r.PathValue("envName")
		env, ok := store.Get(projectUUID, envName)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(env)
	})
	mux.HandleFunc("DELETE /api/v1/projects/{projectUUID}/environments/{envName}", func(w http.ResponseWriter, r *http.Request) {
		projectUUID := r.PathValue("projectUUID")
		envName := r.PathValue("envName")
		store.Delete(projectUUID, envName)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "readback-failure"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Environment created but refresh failed.*Could not read environment.*partial Terraform state was saved`),
			},
		},
	})
}

func TestEnvironmentResource_UpdateDescription(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "update-env"
  description  = "initial"
}
`,
				Check: resource.TestCheckResourceAttr("coolify_environment.test", "description", "initial"),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "update-env"
  description  = "updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment.test", "name", "update-env"),
					resource.TestCheckResourceAttr("coolify_environment.test", "description", "updated"),
				),
			},
		},
	})
}

func TestEnvironmentResource_Import(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "import-env"
  description  = "import test"
}
`,
			},
			{
				ResourceName:            "coolify_environment.test",
				ImportState:             true,
				ImportStateId:           "aaaa0001-0001-4000-8000-000000000001:import-env",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"description"},
			},
		},
	})
}

func TestEnvironmentResource_ImportAndDestroyWithEscapedName(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkEnvironmentDestroy(server.URL),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "qa/staging"
  description  = "slash name"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment.test", "name", "qa/staging"),
					resource.TestCheckResourceAttr("coolify_environment.test", "description", "slash name"),
				),
			},
			{
				ResourceName:            "coolify_environment.test",
				ImportState:             true,
				ImportStateId:           "aaaa0001-0001-4000-8000-000000000001:qa/staging",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"description"},
			},
		},
	})
}

func TestEnvironmentResource_Disappears(t *testing.T) {
	t.Parallel()
	server, store := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "disappearing-env"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment.test", "id"),
					func(s *terraform.State) error {
						store.Delete("aaaa0001-0001-4000-8000-000000000001", "disappearing-env")
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
