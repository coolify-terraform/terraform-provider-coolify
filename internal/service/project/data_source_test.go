package project_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProjectDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "source" {
  name        = "ds-test-project"
  description = "data source test"
}

data "coolify_project" "test" {
  uuid = coolify_project.source.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coolify_project.test", "uuid",
						"coolify_project.source", "uuid",
					),
					resource.TestCheckResourceAttr("data.coolify_project.test", "name", "ds-test-project"),
					resource.TestCheckResourceAttr("data.coolify_project.test", "description", "data source test"),
				),
			},
		},
	})
}

func TestProjectDataSource_NotFound(t *testing.T) {
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
data "coolify_project" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`Error reading project`),
			},
		},
	})
}
