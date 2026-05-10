package environment_test

import (
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
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
				Config: providerConfig(server.URL) + `
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
		},
	})
}
