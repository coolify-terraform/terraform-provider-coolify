package environment_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "source" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "ds-test-env"
  description  = "data source test"
}

data "coolify_environment" "test" {
  project_uuid = coolify_environment.source.project_uuid
  name         = coolify_environment.source.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coolify_environment.test", "id",
						"coolify_environment.source", "id",
					),
					resource.TestCheckResourceAttr("data.coolify_environment.test", "name", "ds-test-env"),
					resource.TestCheckResourceAttr("data.coolify_environment.test", "description", ""),
				),
			},
		},
	})
}

func TestEnvironmentDataSource_NotFound(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_environment" "test" {
  project_uuid = "00000000-0000-4000-8000-000000000000"
  name         = "nonexistent"
}`,
				ExpectError: regexp.MustCompile(`Error reading environment`),
			},
		},
	})
}
