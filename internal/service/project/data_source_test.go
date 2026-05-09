package project_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProjectDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
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
