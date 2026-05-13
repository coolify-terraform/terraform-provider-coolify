package environment_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEnvironmentResource_CreateImport(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-env")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_environment" "test" {
  project_uuid = coolify_project.test.uuid
  name         = "staging"
  description  = "Staging environment"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment.test", "id"),
					resource.TestCheckResourceAttr("coolify_environment.test", "name", "staging"),
					resource.TestCheckResourceAttr("coolify_environment.test", "description", "Staging environment"),
				),
			},
			// Import
			{
				ResourceName:      "coolify_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["coolify_environment.test"]
					return rs.Primary.Attributes["project_uuid"] + ":" + rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func TestAccEnvironmentDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-env-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_environment" "test" {
  project_uuid = coolify_project.test.uuid
  name         = "ds-test"
  description  = "data source test"
}

data "coolify_environment" "test" {
  project_uuid = coolify_project.test.uuid
  name         = coolify_environment.test.name
}

data "coolify_environments" "all" {
  project_uuid = coolify_project.test.uuid
  depends_on   = [coolify_environment.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Single data source
					resource.TestCheckResourceAttrPair(
						"data.coolify_environment.test", "id",
						"coolify_environment.test", "id",
					),
					resource.TestCheckResourceAttr("data.coolify_environment.test", "name", "ds-test"),
					resource.TestCheckResourceAttr("data.coolify_environment.test", "description", "data source test"),
					// List data source
					resource.TestCheckResourceAttrSet("data.coolify_environments.all", "environments.#"),
				),
			},
		},
	})
}
