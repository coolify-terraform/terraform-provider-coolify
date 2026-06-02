package githubapp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGitHubAppsListDataSource(t *testing.T) {
	t.Parallel()
	apps := []client.GitHubApp{
		{
			ID:               1,
			UUID:             "ghapp-list-uuid-1",
			Name:             "my-app",
			OrganizationName: "my-org",
			AppID:            12345,
			InstallationID:   67890,
			ClientID:         "Iv1.abc123",
			WebhookSecret:    "whsec-secret",
		},
		{
			ID:               2,
			UUID:             "ghapp-list-uuid-2",
			Name:             "other-app",
			OrganizationName: "",
			AppID:            54321,
			InstallationID:   98765,
			ClientID:         "Iv1.def456",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/github-apps" {
			json.NewEncoder(w).Encode(apps)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_github_apps" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.id", "1"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.uuid", "ghapp-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.name", "my-app"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.organization_name", "my-org"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.app_id", "12345"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.installation_id", "67890"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.0.client_id", "Iv1.abc123"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.1.id", "2"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.1.uuid", "ghapp-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.1.name", "other-app"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.test", "github_apps.1.app_id", "54321"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_github_apps" "filtered" {
  filter {
    name   = "name"
    values = ["my-app"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_github_apps.filtered", "github_apps.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_github_apps.filtered", "github_apps.0.name", "my-app"),
				),
			},
		},
	})
}

func TestGitHubAppsListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_github_apps" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing GitHub Apps`),
			},
		},
	})
}
