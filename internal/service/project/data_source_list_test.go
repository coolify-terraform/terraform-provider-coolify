package project_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProjectListDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "first" {
  name        = "first-project"
  description = "first"
}

resource "coolify_project" "second" {
  name        = "second-project"
  description = "second"
}

data "coolify_projects" "all" {
  depends_on = [coolify_project.first, coolify_project.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_projects.all", "projects.#", "2"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_project" "first" {
  name        = "first-project"
  description = "first"
}

resource "coolify_project" "second" {
  name        = "second-project"
  description = "second"
}

data "coolify_projects" "filtered" {
  depends_on = [coolify_project.first, coolify_project.second]
  filter {
    name   = "name"
    values = ["first-project"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_projects.filtered", "projects.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_projects.filtered", "projects.0.name", "first-project"),
				),
			},
		},
	})
}
