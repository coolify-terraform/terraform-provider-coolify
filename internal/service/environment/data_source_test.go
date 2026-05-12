package environment_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
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
