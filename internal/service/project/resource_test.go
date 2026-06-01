package project_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/spectest"
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
func newMockCoolifyServer(auditT ...testing.TB) (*httptest.Server, *mockProjectStore) {
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
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("DELETE /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if !store.Delete(uuid) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("GET /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		projects := store.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	})

	handler := acctest.WithVersionEndpoint(mux)
	if len(auditT) > 0 {
		handler = spectest.WithSpecValidation(auditT[0], "coolify-v4", handler)
	}
	server := httptest.NewServer(handler)
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
		UUID:        fmt.Sprintf("cccc0000-0000-4000-8000-%012d", s.counter),
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

func TestProjectResource_Create(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(server.URL, "coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

func TestProjectResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	store := &mockProjectStore{
		projects: make(map[string]*mockProject),
		counter:  0,
	}

	var forceReadFailure atomic.Bool

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
		forceReadFailure.Store(true)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": p.UUID})
	})
	mux.HandleFunc("GET /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if forceReadFailure.Load() {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
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

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "test" {
  name = "readback-failure"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Project created but refresh failed.*Could not read project.*partial Terraform state was saved`),
			},
		},
	})
}

func TestProjectResource_Update(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

func TestProjectResource_UpdateUsesPatchResponse(t *testing.T) {
	t.Parallel()
	store := &mockProjectStore{
		projects: make(map[string]*mockProject),
		counter:  0,
	}

	var countStepTwoRequests atomic.Bool
	var stepTwoPatches atomic.Int32
	var stepTwoGetsAfterPatch atomic.Int32

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
		if countStepTwoRequests.Load() && stepTwoPatches.Load() > 0 {
			stepTwoGetsAfterPatch.Add(1)
		}
		uuid := r.PathValue("uuid")
		p, ok := store.Get(uuid)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(p)
	})
	mux.HandleFunc("PATCH /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if countStepTwoRequests.Load() {
			stepTwoPatches.Add(1)
		}
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
		store.Delete(uuid)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})
	mux.HandleFunc("GET /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(store.List())
	})

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
				PreConfig: func() {
					stepTwoPatches.Store(0)
					stepTwoGetsAfterPatch.Store(0)
					countStepTwoRequests.Store(true)
				},
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "test" {
  name        = "updated-name"
  description = "updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_project.test", "name", "updated-name"),
					resource.TestCheckResourceAttr("coolify_project.test", "description", "updated description"),
					func(_ *terraform.State) error {
						countStepTwoRequests.Store(false)
						if got := stepTwoPatches.Load(); got != 1 {
							return fmt.Errorf("expected 1 PATCH during update step, got %d", got)
						}
						if got := stepTwoGetsAfterPatch.Load(); got != 0 {
							return fmt.Errorf("expected 0 GETs after PATCH during update step, got %d", got)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestProjectResource_Import(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

func TestProjectResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "test" {
  name        = "import-project"
  description = "import test"
}
`,
			},
			{
				ResourceName:  "coolify_project.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestProjectResource_Disappears(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
	t.Parallel()
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

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

// TestProjectResource_SpecValidation runs the full CRUD lifecycle against
// a mock server that validates every request/response against the OpenAPI spec.
func TestProjectResource_SpecValidation(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer(t)
	defer server.Close()

	name := acctest.RandomWithPrefix("tf-spec")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + fmt.Sprintf(`
resource "coolify_project" "test" {
  name        = %q
  description = "spec validation test"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_project.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_project.test", "name", name),
				),
			},
		},
	})
}

// TestProjectResource_DeleteRetry verifies that project deletion retries
// on "has resources" errors (Coolify's async child-resource cleanup).
func TestProjectResource_DeleteRetry(t *testing.T) {
	t.Parallel()
	var deleteAttempts atomic.Int32

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
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}
		p := store.Create(body.Name, body.Description)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": p.UUID})
	})
	mux.HandleFunc("GET /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := store.Get(r.PathValue("uuid"))
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(p)
	})
	mux.HandleFunc("DELETE /api/v1/projects/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		attempt := deleteAttempts.Add(1)
		if attempt <= 2 {
			http.Error(w, `{"message":"Project has resources, so it cannot be deleted. Delete the resources first."}`, http.StatusUnprocessableEntity)
			return
		}
		uuid := r.PathValue("uuid")
		store.Delete(uuid)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})
	mux.HandleFunc("GET /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(store.List())
	})

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(s *terraform.State) error {
			if got := int(deleteAttempts.Load()); got < 3 {
				return fmt.Errorf("expected at least 3 delete attempts, got %d", got)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "test" {
  name = "retry-test-project"
}
`,
				Check: resource.TestCheckResourceAttrSet("coolify_project.test", "uuid"),
			},
		},
	})
}
