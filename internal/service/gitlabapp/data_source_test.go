package gitlabapp_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGitLabAppDataSource_Read(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/gitlab-apps", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"id":7,"uuid":"gggg0001-0001-4000-8000-000000000001","name":"corp-gitlab","html_url":"https://gitlab.example.com","api_url":"https://gitlab.example.com/api/v4"}
		]`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitlab_app" "by_id" {
  id = 7
}

data "coolify_gitlab_app" "by_uuid" {
  uuid = "gggg0001-0001-4000-8000-000000000001"
}

data "coolify_gitlab_apps" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_gitlab_app.by_id", "name", "corp-gitlab"),
					resource.TestCheckResourceAttr("data.coolify_gitlab_app.by_id", "html_url", "https://gitlab.example.com"),
					resource.TestCheckResourceAttr("data.coolify_gitlab_app.by_uuid", "id", "7"),
					resource.TestCheckResourceAttr("data.coolify_gitlab_apps.all", "apps.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_gitlab_apps.all", "apps.0.name", "corp-gitlab"),
				),
			},
		},
	})
}

func TestGitLabAppDataSource_ReadAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/gitlab-apps", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitlab_app" "test" {
  id = 7
}
`,
			ExpectError: regexp.MustCompile(`Error reading GitLab App`),
		}},
	})
}

func TestGitLabAppsListDataSource_ReadAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/gitlab-apps", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitlab_apps" "all" {}
`,
			ExpectError: regexp.MustCompile(`Error listing GitLab Apps`),
		}},
	})
}

func TestGitLabAppDataSource_MissingLookup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NewServeMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitlab_app" "test" {}
`,
			ExpectError: regexp.MustCompile(`set id or uuid`),
		}},
	})
}
