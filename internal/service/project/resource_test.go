package project_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// mockProject stores project data in the mock server.
type mockProject struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// newMockCoolifyServer creates an httptest.Server that simulates the Coolify API for projects.
func newMockCoolifyServer() (*httptest.Server, *mockProjectStore) {
	store := &mockProjectStore{
		projects: make(map[string]*mockProject),
		counter:  0,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		p := store.Create(body.Name, body.Description)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": p.UUID})
	})

	mux.HandleFunc("GET /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		p, ok := store.Get(uuid)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("PATCH /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		p, ok := store.Update(uuid, body.Name, body.Description)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("DELETE /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if !store.Delete(uuid) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("GET /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		projects := store.List()
		json.NewEncoder(w).Encode(projects)
	})

	server := httptest.NewServer(mux)
	return server, store
}

// mockProjectStore is a thread-safe in-memory store for mock projects.
type mockProjectStore struct {
	mu       sync.RWMutex
	projects map[string]*mockProject
	counter  int
}

func (s *mockProjectStore) Create(name, description string) *mockProject {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	p := &mockProject{
		UUID:        fmt.Sprintf("test-uuid-%d", s.counter),
		Name:        name,
		Description: description,
	}
	s.projects[p.UUID] = p
	return p
}

func (s *mockProjectStore) Get(uuid string) (*mockProject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[uuid]
	return p, ok
}

func (s *mockProjectStore) Update(uuid, name, description string) (*mockProject, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[uuid]
	if !ok {
		return nil, false
	}
	p.Name = name
	p.Description = description
	return p, true
}

func (s *mockProjectStore) Delete(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.projects[uuid]
	if ok {
		delete(s.projects, uuid)
	}
	return ok
}

func (s *mockProjectStore) List() []*mockProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*mockProject, 0, len(s.projects))
	for _, p := range s.projects {
		result = append(result, p)
	}
	return result
}

// testProviderFactory returns a provider factory wired to the given server URL.
func testProviderFactory(serverURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// providerConfig returns the HCL provider block pointing at the mock server.
func providerConfig(serverURL string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}
`, serverURL)
}

func TestProjectResource_Create(t *testing.T) {
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name        = "my-project"
  description = "A test project"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_project.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_project.test", "name", "my-project"),
					resource.TestCheckResourceAttr("coolify_project.test", "description", "A test project"),
				),
			},
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name        = "my-project"
  description = "A test project"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestProjectResource_Update(t *testing.T) {
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name        = "original-name"
  description = "original description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_project.test", "name", "original-name"),
					resource.TestCheckResourceAttr("coolify_project.test", "description", "original description"),
				),
			},
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name        = "updated-name"
  description = "updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_project.test", "name", "updated-name"),
					resource.TestCheckResourceAttr("coolify_project.test", "description", "updated description"),
				),
			},
		},
	})
}

func TestProjectResource_Import(t *testing.T) {
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name        = "import-project"
  description = "import test"
}
`,
			},
			{
				ResourceName:                         "coolify_project.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_project.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
		},
	})
}

func TestProjectResource_Disappears(t *testing.T) {
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name = "disappearing-project"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_project.test", "uuid"),
					// Delete the project out-of-band to simulate external deletion.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_project.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						store.Delete(uuid)
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestProjectResource_ReadNotFound(t *testing.T) {
	store := &mockProjectStore{
		projects: make(map[string]*mockProject),
		counter:  0,
	}

	var forceNotFound atomic.Bool

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		p := store.Create(body.Name, body.Description)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": p.UUID})
	})

	mux.HandleFunc("GET /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if forceNotFound.Load() {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		uuid := r.PathValue("uuid")
		p, ok := store.Get(uuid)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("DELETE /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		store.Delete(uuid)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name = "notfound-project"
}
`,
				Check: resource.TestCheckResourceAttrSet("coolify_project.test", "uuid"),
			},
			{
				PreConfig: func() {
					forceNotFound.Store(true)
				},
				Config: providerConfig(server.URL) + `
resource "coolify_project" "test" {
  name = "notfound-project"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
