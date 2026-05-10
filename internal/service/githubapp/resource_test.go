package githubapp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// mockGitHubApp stores GitHub App data in the mock server.
type mockGitHubApp struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization_name,omitempty"`
	AppID            int64  `json:"app_id,omitempty"`
	InstallationID   int64  `json:"installation_id,omitempty"`
	ClientID         string `json:"client_id,omitempty"`
	WebhookSecret    string `json:"webhook_secret,omitempty"`
}

// mockGitHubAppStore is a thread-safe in-memory store for mock GitHub Apps.
type mockGitHubAppStore struct {
	mu      sync.Mutex
	apps    map[int64]*mockGitHubApp
	counter int64
}

func (s *mockGitHubAppStore) Create(name, orgName string, appID, installID int64, clientID, webhookSecret string) *mockGitHubApp {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	app := &mockGitHubApp{
		ID:               s.counter,
		Name:             name,
		OrganizationName: orgName,
		AppID:            appID,
		InstallationID:   installID,
		ClientID:         clientID,
		WebhookSecret:    webhookSecret,
	}
	s.apps[app.ID] = app
	return app
}

func (s *mockGitHubAppStore) Get(id int64) (*mockGitHubApp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.apps[id]
	return app, ok
}

func (s *mockGitHubAppStore) Update(id int64, name *string, webhookSecret *string) (*mockGitHubApp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.apps[id]
	if !ok {
		return nil, false
	}
	if name != nil {
		app.Name = *name
	}
	if webhookSecret != nil {
		app.WebhookSecret = *webhookSecret
	}
	return app, true
}

func (s *mockGitHubAppStore) Delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.apps[id]
	if ok {
		delete(s.apps, id)
	}
	return ok
}

func (s *mockGitHubAppStore) List() []*mockGitHubApp {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*mockGitHubApp, 0, len(s.apps))
	for _, a := range s.apps {
		result = append(result, a)
	}
	return result
}

// newMockCoolifyServer creates an httptest.Server that simulates the Coolify API for GitHub Apps.
func newMockCoolifyServer() (*httptest.Server, *mockGitHubAppStore) {
	store := &mockGitHubAppStore{
		apps:    make(map[int64]*mockGitHubApp),
		counter: 0,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/github-apps", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name             string `json:"name"`
			OrganizationName string `json:"organization_name"`
			AppID            int64  `json:"app_id"`
			InstallationID   int64  `json:"installation_id"`
			ClientID         string `json:"client_id"`
			ClientSecret     string `json:"client_secret"`
			WebhookSecret    string `json:"webhook_secret"`
			PrivateKey       string `json:"private_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		app := store.Create(body.Name, body.OrganizationName, body.AppID, body.InstallationID, body.ClientID, body.WebhookSecret)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(app)
	})

	mux.HandleFunc("GET /api/v1/github-apps", func(w http.ResponseWriter, r *http.Request) {
		apps := store.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apps)
	})

	mux.HandleFunc("PATCH /api/v1/github-apps/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}

		var body struct {
			Name          *string `json:"name"`
			WebhookSecret *string `json:"webhook_secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		app, ok := store.Update(id, body.Name, body.WebhookSecret)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	mux.HandleFunc("DELETE /api/v1/github-apps/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}

		if !store.Delete(id) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("GET /api/v1/github-apps/{id}/repositories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.GitHubRepository{
			{Name: "repo-1", FullName: "testowner/repo-1", Private: false},
			{Name: "repo-2", FullName: "testowner/repo-2", Private: true},
		})
	})

	mux.HandleFunc("GET /api/v1/github-apps/{id}/repositories/{owner}/{repo}/branches", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.GitHubBranch{
			{Name: "main"},
			{Name: "develop"},
		})
	})

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	return server, store
}

func testProviderBlock(serverURL string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}
`, serverURL)
}

func TestGitHubAppResource_Create(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "my-github-app"
  app_id          = 12345
  installation_id = 67890
  client_id       = "Iv1.abc123"
  client_secret   = "secret123"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "name", "my-github-app"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "app_id", "12345"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "installation_id", "67890"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "client_id", "Iv1.abc123"),
				),
			},
			// Plan idempotency
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "my-github-app"
  app_id          = 12345
  installation_id = 67890
  client_id       = "Iv1.abc123"
  client_secret   = "secret123"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestGitHubAppResource_Import(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "import-app"
  app_id          = 11111
  installation_id = 22222
  client_id       = "Iv1.import"
  client_secret   = "importsecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\nimport\n-----END RSA PRIVATE KEY-----"
}
`,
				Check: resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
			},
			{
				ResourceName:                         "coolify_github_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"client_secret", "private_key"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_github_app.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func TestGitHubAppResource_Disappears(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "disappearing-app"
  app_id          = 99999
  installation_id = 88888
  client_id       = "Iv1.disappear"
  client_secret   = "disappearsecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\ndisappear\n-----END RSA PRIVATE KEY-----"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
					// Delete the GitHub App out-of-band to simulate external deletion.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_github_app.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						idStr := rs.Primary.Attributes["id"]
						id, _ := strconv.ParseInt(idStr, 10, 64)
						store.Delete(id)
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestGitHubAppsDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "first" {
  name            = "first-app"
  app_id          = 111
  installation_id = 222
  client_id       = "Iv1.first"
  client_secret   = "firstsecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\nfirst\n-----END RSA PRIVATE KEY-----"
}

resource "coolify_github_app" "second" {
  name            = "second-app"
  app_id          = 333
  installation_id = 444
  client_id       = "Iv1.second"
  client_secret   = "secondsecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\nsecond\n-----END RSA PRIVATE KEY-----"
}

data "coolify_github_apps" "all" {
  depends_on = [coolify_github_app.first, coolify_github_app.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_apps.all", "github_apps.#", "2"),
				),
			},
		},
	})
}

func TestGitHubAppRepositoriesDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "repos-test-app"
  app_id          = 555
  installation_id = 666
  client_id       = "Iv1.repos"
  client_secret   = "repossecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\nrepos\n-----END RSA PRIVATE KEY-----"
}

data "coolify_github_app_repositories" "test" {
  github_app_id = coolify_github_app.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.0.name", "repo-1"),
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.0.full_name", "testowner/repo-1"),
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.0.private", "false"),
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.1.name", "repo-2"),
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.1.full_name", "testowner/repo-2"),
					resource.TestCheckResourceAttr("data.coolify_github_app_repositories.test", "repositories.1.private", "true"),
				),
			},
		},
	})
}

func TestGitHubAppBranchesDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "branches-test-app"
  app_id          = 777
  installation_id = 888
  client_id       = "Iv1.branches"
  client_secret   = "branchessecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\nbranches\n-----END RSA PRIVATE KEY-----"
}

data "coolify_github_app_branches" "test" {
  github_app_id = coolify_github_app.test.id
  owner         = "testowner"
  repo          = "repo-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_app_branches.test", "branches.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_github_app_branches.test", "branches.0.name", "main"),
					resource.TestCheckResourceAttr("data.coolify_github_app_branches.test", "branches.1.name", "develop"),
				),
			},
		},
	})
}
