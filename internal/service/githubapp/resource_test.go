package githubapp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/githubapp"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// mockGitHubApp stores GitHub App data in the mock server.
type mockGitHubApp struct {
	ID               int64  `json:"id"`
	UUID             string `json:"uuid,omitempty"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization,omitempty"`
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
		UUID:             fmt.Sprintf("ghapp-uuid-%03d", s.counter),
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

func (s *mockGitHubAppStore) Update(id int64, upd mockGitHubAppUpdate) (*mockGitHubApp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.apps[id]
	if !ok {
		return nil, false
	}
	if upd.Name != nil {
		app.Name = *upd.Name
	}
	if upd.OrganizationName != nil {
		app.OrganizationName = *upd.OrganizationName
	}
	if upd.AppID != nil {
		app.AppID = *upd.AppID
	}
	if upd.InstallationID != nil {
		app.InstallationID = *upd.InstallationID
	}
	if upd.ClientID != nil {
		app.ClientID = *upd.ClientID
	}
	if upd.WebhookSecret != nil {
		app.WebhookSecret = *upd.WebhookSecret
	}
	return app, true
}

type mockGitHubAppUpdate struct {
	Name             *string `json:"name"`
	OrganizationName *string `json:"organization"`
	AppID            *int64  `json:"app_id"`
	InstallationID   *int64  `json:"installation_id"`
	ClientID         *string `json:"client_id"`
	WebhookSecret    *string `json:"webhook_secret"`
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

func requireMockGitHubApp(w http.ResponseWriter, r *http.Request, store *mockGitHubAppStore) bool {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return false
	}

	if _, ok := store.Get(id); !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return false
	}

	return true
}

// newMockCoolifyServer creates an httptest.Server that simulates the Coolify API for GitHub Apps.
func newMockCoolifyServer(auditT ...testing.TB) (*httptest.Server, *mockGitHubAppStore) {
	store := &mockGitHubAppStore{
		apps: make(map[int64]*mockGitHubApp),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/github-apps", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name             string `json:"name"`
			OrganizationName string `json:"organization"`
			APIURL           string `json:"api_url"`
			HTMLURL          string `json:"html_url"`
			AppID            int64  `json:"app_id"`
			InstallationID   int64  `json:"installation_id"`
			ClientID         string `json:"client_id"`
			ClientSecret     string `json:"client_secret"`
			WebhookSecret    string `json:"webhook_secret"`
			PrivateKeyUUID   string `json:"private_key_uuid"`
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

		var body mockGitHubAppUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		app, ok := store.Update(id, body)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"message": "GitHub app updated successfully", "data": app})
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
		if !requireMockGitHubApp(w, r, store) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"repositories": []client.GitHubRepository{
			{Name: "repo-1", FullName: "testowner/repo-1", Private: false},
			{Name: "repo-2", FullName: "testowner/repo-2", Private: true},
		}})
	})

	mux.HandleFunc("GET /api/v1/github-apps/{id}/repositories/{owner}/{repo}/branches", func(w http.ResponseWriter, r *http.Request) {
		if !requireMockGitHubApp(w, r, store) {
			return
		}
		if r.PathValue("owner") != "testowner" || r.PathValue("repo") != "repo-1" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"branches": []client.GitHubBranch{
			{Name: "main"},
			{Name: "develop"},
		}})
	})

	handler := acctest.WithVersionEndpoint(mux)
	if len(auditT) > 0 {
		handler = spectest.WithSpecAudit(auditT[0], "coolify-v4", handler)
	}
	server := httptest.NewServer(handler)
	return server, store
}

func TestGitHubAppResource_Create(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer(t)
	defer server.Close()

	config := testGitHubAppResourceConfig(server.URL, `
name             = "my-github-app"
app_id           = 12345
installation_id  = 67890
client_id        = "Iv1.abc123"
client_secret    = "secret123"
private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkGitHubAppDestroy(server.URL),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "name", "my-github-app"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "app_id", "12345"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "installation_id", "67890"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "client_id", "Iv1.abc123"),
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "webhook_secret"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_github_app.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}

						secret := rs.Primary.Attributes["webhook_secret"]
						if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(secret) {
							return fmt.Errorf("expected generated webhook_secret to be 64 lowercase hex characters, got %q", secret)
						}
						if secret == "my-github-app-webhook" || secret == "terraform-provider-coolify" {
							return fmt.Errorf("expected generated webhook_secret to avoid predictable defaults, got %q", secret)
						}

						id, err := strconv.ParseInt(rs.Primary.Attributes["id"], 10, 64)
						if err != nil {
							return fmt.Errorf("parsing resource id: %w", err)
						}
						app, ok := store.Get(id)
						if !ok {
							return fmt.Errorf("mock github app %d not found", id)
						}
						if app.WebhookSecret != secret {
							return fmt.Errorf("expected API payload webhook_secret to match state")
						}

						return nil
					},
				),
			},
			// Plan idempotency
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestGitHubAppResource_Update(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	initialConfig := testGitHubAppResourceConfig(server.URL, `
name             = "my-github-app"
app_id           = 12345
installation_id  = 67890
client_id        = "Iv1.abc123"
client_secret    = "secret123"
webhook_secret   = "hook-secret-1"
private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
`)
	updatedConfig := testGitHubAppResourceConfig(server.URL, `
name             = "my-github-app-updated"
app_id           = 54321
installation_id  = 99999
client_id        = "Iv1.xyz789"
client_secret    = "secret456"
webhook_secret   = "hook-secret-2"
private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "name", "my-github-app"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "webhook_secret", "hook-secret-1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_github_app.test", "name", "my-github-app-updated"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "app_id", "54321"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "installation_id", "99999"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "client_id", "Iv1.xyz789"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "webhook_secret", "hook-secret-2"),
				),
			},
			// Plan idempotency after update
			{
				Config:             updatedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestGitHubAppResource_CreateUsesCreateResponse(t *testing.T) {
	t.Parallel()

	store := &mockGitHubAppStore{apps: make(map[int64]*mockGitHubApp)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/github-apps", func(w http.ResponseWriter, r *http.Request) {
		var body client.CreateGitHubAppIntegrationInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		createdApp := store.Create(body.Name, body.OrganizationName, body.AppID, body.InstallationID, body.ClientID, body.WebhookSecret)
		responseApp := *createdApp
		responseApp.UUID = "ghapp-create-response"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(responseApp)
	})
	mux.HandleFunc("GET /api/v1/github-apps", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.List())
	})
	mux.HandleFunc("DELETE /api/v1/github-apps/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}
		if !store.Delete(id) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	config := testGitHubAppResourceConfig(server.URL, `
name              = "create-response"
organization_name = "acme"
app_id            = 12345
installation_id   = 67890
client_id         = "Iv1.readback"
client_secret     = "secret123"
webhook_secret    = "hook-secret"
private_key_uuid  = "dddd0001-0001-4000-8000-000000000001"
`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkGitHubAppDestroy(server.URL),
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
				resource.TestCheckResourceAttr("coolify_github_app.test", "uuid", "ghapp-create-response"),
				resource.TestCheckResourceAttr("coolify_github_app.test", "name", "create-response"),
				resource.TestCheckResourceAttr("coolify_github_app.test", "organization_name", "acme"),
				resource.TestCheckResourceAttr("coolify_github_app.test", "webhook_secret", "hook-secret"),
			),
		}},
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "import-app"
  app_id          = 11111
  installation_id = 22222
  client_id       = "Iv1.import"
  client_secret   = "importsecret"
  private_key_uuid = "dddd0003-0003-4000-8000-000000000003"
}
`,
				Check: resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
			},
			{
				ResourceName:                         "coolify_github_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"client_secret", "webhook_secret", "private_key_uuid"},
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "disappearing-app"
  app_id          = 99999
  installation_id = 88888
  client_id       = "Iv1.disappear"
  client_secret   = "disappearsecret"
  private_key_uuid = "dddd0004-0004-4000-8000-000000000004"
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
						id, err := strconv.ParseInt(idStr, 10, 64)
						if err != nil {
							return fmt.Errorf("invalid resource id %q: %w", idStr, err)
						}
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "first" {
  name            = "first-app"
  app_id          = 111
  installation_id = 222
  client_id       = "Iv1.first"
  client_secret   = "firstsecret"
  private_key_uuid = "dddd0005-0005-4000-8000-000000000005"
}

resource "coolify_github_app" "second" {
  name            = "second-app"
  app_id          = 333
  installation_id = 444
  client_id       = "Iv1.second"
  client_secret   = "secondsecret"
  private_key_uuid = "dddd0006-0006-4000-8000-000000000006"
}

data "coolify_github_apps" "all" {
  depends_on = [coolify_github_app.first, coolify_github_app.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_apps.all", "github_apps.#", "2"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "first" {
  name            = "first-app"
  app_id          = 111
  installation_id = 222
  client_id       = "Iv1.first"
  client_secret   = "firstsecret"
  private_key_uuid = "dddd0005-0005-4000-8000-000000000005"
}

resource "coolify_github_app" "second" {
  name            = "second-app"
  app_id          = 333
  installation_id = 444
  client_id       = "Iv1.second"
  client_secret   = "secondsecret"
  private_key_uuid = "dddd0006-0006-4000-8000-000000000006"
}

data "coolify_github_apps" "filtered" {
  depends_on = [coolify_github_app.first, coolify_github_app.second]
  filter {
    name   = "name"
    values = ["second-app"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_apps.filtered", "github_apps.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.filtered", "github_apps.0.name", "second-app"),
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "repos-test-app"
  app_id          = 555
  installation_id = 666
  client_id       = "Iv1.repos"
  client_secret   = "repossecret"
  private_key_uuid = "dddd0007-0007-4000-8000-000000000007"
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "branches-test-app"
  app_id          = 777
  installation_id = 888
  client_id       = "Iv1.branches"
  client_secret   = "branchessecret"
  private_key_uuid = "dddd0008-0008-4000-8000-000000000008"
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

func TestGitHubAppDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "ds-test-app"
  app_id          = 11111
  installation_id = 22222
  client_id       = "Iv1.dstest"
  client_secret   = "dstestsecret"
  private_key_uuid = "dddd0009-0009-4000-8000-000000000009"
}

data "coolify_github_app" "test" {
  id = coolify_github_app.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_github_app.test", "id"),
					resource.TestCheckResourceAttr("data.coolify_github_app.test", "name", "ds-test-app"),
					resource.TestCheckResourceAttr("data.coolify_github_app.test", "app_id", "11111"),
					resource.TestCheckResourceAttr("data.coolify_github_app.test", "installation_id", "22222"),
					resource.TestCheckResourceAttr("data.coolify_github_app.test", "client_id", "Iv1.dstest"),
				),
			},
		},
	})
}

func TestGitHubAppDataSource_NotFound(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
data "coolify_github_app" "test" {
  id = 999
}
`,
				ExpectError: regexp.MustCompile("Error reading GitHub App"),
			},
		},
	})
}

func TestGitHubAppsDataSource_ClientError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_github_apps" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing GitHub Apps`),
			},
		},
	})
}

func TestGitHubAppRepositoriesDataSource_ClientError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_github_app_repositories" "test" {
  github_app_id = 123
}
`,
				ExpectError: regexp.MustCompile(`Error listing repositories`),
			},
		},
	})
}

func TestGitHubAppBranchesDataSource_ClientError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_github_app_branches" "test" {
  github_app_id = 123
  owner         = "testowner"
  repo          = "testrepo"
}
`,
				ExpectError: regexp.MustCompile(`Error listing branches`),
			},
		},
	})
}

func TestGitHubAppResource_ImportBadID(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_github_app" "test" {
  name            = "import-bad-id"
  app_id          = 12345
  installation_id = 67890
  client_id       = "Iv1.badid"
  client_secret   = "badidsecret"
  private_key_uuid = "dddd0010-0010-4000-8000-000000000010"
}
`,
			},
			{
				ResourceName:  "coolify_github_app.test",
				ImportState:   true,
				ImportStateId: "not-a-number",
				ExpectError:   regexp.MustCompile("Invalid Import ID"),
			},
		},
	})
}

func testGitHubAppResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_github_app", "test", attrs)
}

// checkGitHubAppDestroy verifies that all coolify_github_app resources have
// been removed from the mock server. The standard acctest.CheckDestroy helper
// looks up by "uuid", but GitHub Apps use a numeric "id" attribute instead.
func checkGitHubAppDestroy(serverURL string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "coolify_github_app" {
				continue
			}
			idStr := rs.Primary.Attributes["id"]
			if idStr == "" {
				continue
			}
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q: %w", idStr, err)
			}
			resp, err := http.Get(serverURL + "/api/v1/github-apps")
			if err != nil {
				return fmt.Errorf("error checking destroy for coolify_github_app/%s: %w", idStr, err)
			}
			var apps []mockGitHubApp
			decodeErr := json.NewDecoder(resp.Body).Decode(&apps)
			resp.Body.Close()
			if decodeErr != nil {
				return fmt.Errorf("decoding GitHub Apps destroy check response: %w", decodeErr)
			}
			for _, app := range apps {
				if app.ID == id {
					return fmt.Errorf("coolify_github_app %s still exists", idStr)
				}
			}
		}
		return nil
	}
}

func TestGitHubAppResource_UpgradeStateV0(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	res := githubapp.NewResource()

	// Get current (v1) schema.
	var schemaResp fwresource.SchemaResponse
	res.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	// Get v0 upgrader.
	upgraders := res.(fwresource.ResourceWithUpgradeState).UpgradeState(ctx)
	v0Up, ok := upgraders[0]
	if !ok {
		t.Fatal("v0 state upgrader not found")
	}

	// Build v0 raw state with the old "private_key" field (raw PEM content).
	v0Raw := tftypes.NewValue(
		v0Up.PriorSchema.Type().TerraformType(ctx),
		map[string]tftypes.Value{
			"id":                tftypes.NewValue(tftypes.Number, 42),
			"uuid":              tftypes.NewValue(tftypes.String, "gh-app-uuid-001"),
			"name":              tftypes.NewValue(tftypes.String, "my-github-app"),
			"organization_name": tftypes.NewValue(tftypes.String, "my-org"),
			"app_id":            tftypes.NewValue(tftypes.Number, 111),
			"installation_id":   tftypes.NewValue(tftypes.Number, 222),
			"client_id":         tftypes.NewValue(tftypes.String, "Iv1.test"),
			"client_secret":     tftypes.NewValue(tftypes.String, "secret-value"),
			"webhook_secret":    tftypes.NewValue(tftypes.String, "webhook-secret"),
			"private_key":       tftypes.NewValue(tftypes.String, "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"),
		},
	)
	priorState := tfsdk.State{
		Schema: *v0Up.PriorSchema,
		Raw:    v0Raw,
	}

	// Prepare empty v1 state for the upgrader to populate.
	newState := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}

	req := fwresource.UpgradeStateRequest{State: &priorState}
	resp := fwresource.UpgradeStateResponse{State: newState}
	v0Up.StateUpgrader(ctx, req, &resp)

	// The upgrader should NOT produce errors (it emits a warning instead).
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
	}

	// It should produce a warning about the private_key -> private_key_uuid rename.
	warnings := resp.Diagnostics.Warnings()
	if len(warnings) == 0 {
		t.Fatal("expected a warning about private_key rename, got none")
	}
	found := false
	for _, w := range warnings {
		if w.Summary() == "State migrated: private_key renamed to private_key_uuid" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about private_key rename, got: %v", warnings)
	}

	// Verify core fields were preserved.
	var name, uuid, clientID, clientSecret, webhookSecret, orgName types.String
	var id, appID, installationID types.Int64
	for _, tc := range []struct {
		name string
		dest any
	}{
		{"name", &name},
		{"uuid", &uuid},
		{"client_id", &clientID},
		{"client_secret", &clientSecret},
		{"webhook_secret", &webhookSecret},
		{"organization_name", &orgName},
		{"id", &id},
		{"app_id", &appID},
		{"installation_id", &installationID},
	} {
		if diags := resp.State.GetAttribute(ctx, path.Root(tc.name), tc.dest); diags.HasError() {
			t.Fatalf("failed to get state attribute %q: %v", tc.name, diags)
		}
	}

	if name.ValueString() != "my-github-app" {
		t.Errorf("name = %q, want %q", name.ValueString(), "my-github-app")
	}
	if uuid.ValueString() != "gh-app-uuid-001" {
		t.Errorf("uuid = %q, want %q", uuid.ValueString(), "gh-app-uuid-001")
	}
	if clientID.ValueString() != "Iv1.test" {
		t.Errorf("client_id = %q, want %q", clientID.ValueString(), "Iv1.test")
	}
	if clientSecret.ValueString() != "secret-value" {
		t.Errorf("client_secret = %q, want %q", clientSecret.ValueString(), "secret-value")
	}
	if webhookSecret.ValueString() != "webhook-secret" {
		t.Errorf("webhook_secret = %q, want %q", webhookSecret.ValueString(), "webhook-secret")
	}
	if orgName.ValueString() != "my-org" {
		t.Errorf("organization_name = %q, want %q", orgName.ValueString(), "my-org")
	}
	if id.ValueInt64() != 42 {
		t.Errorf("id = %d, want 42", id.ValueInt64())
	}
	if appID.ValueInt64() != 111 {
		t.Errorf("app_id = %d, want 111", appID.ValueInt64())
	}
	if installationID.ValueInt64() != 222 {
		t.Errorf("installation_id = %d, want 222", installationID.ValueInt64())
	}

	// private_key_uuid should be Unknown (can't convert raw PEM to UUID).
	var pkUUID types.String
	if diags := resp.State.GetAttribute(ctx, path.Root("private_key_uuid"), &pkUUID); diags.HasError() {
		t.Fatalf("failed to get state attribute %q: %v", "private_key_uuid", diags)
	}
	if !pkUUID.IsUnknown() {
		t.Errorf("private_key_uuid should be Unknown after migration, got %q", pkUUID.ValueString())
	}
}
