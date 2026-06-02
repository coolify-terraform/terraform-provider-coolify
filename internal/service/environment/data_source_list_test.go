package environment_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentListDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "first" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "first-env"
  description  = "first"
}

resource "coolify_environment" "second" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "second-env"
  description  = "second"
}

data "coolify_environments" "all" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  depends_on   = [coolify_environment.first, coolify_environment.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environments.all", "environments.#", "2"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "first" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "first-env"
  description  = "first"
}

resource "coolify_environment" "second" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "second-env"
  description  = "second"
}

data "coolify_environments" "filtered" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  depends_on   = [coolify_environment.first, coolify_environment.second]
  filter {
    name   = "name"
    values = ["first-env"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environments.filtered", "environments.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_environments.filtered", "environments.0.name", "first-env"),
				),
			},
		},
	})
}

func TestEnvironmentListDataSource_APIError(t *testing.T) {
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
data "coolify_environments" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`Error listing environments`),
			},
		},
	})
}
