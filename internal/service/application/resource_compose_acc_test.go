package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDockerComposeApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-compose")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDockerComposeAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_docker_compose_application.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_docker_compose_application.test", "name"),
				),
			},
			{
				Config: testAccDockerComposeAppConfig(name, serverUUID, `description = "Updated compose app"`),
				Check:  resource.TestCheckResourceAttr("coolify_docker_compose_application.test", "description", "Updated compose app"),
			},
			{
				ResourceName:                         "coolify_docker_compose_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

func testAccDockerComposeAppConfig(name, serverUUID, extra string) string {
	compose := `services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"80:80\"`
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_docker_compose_application" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  name             = %[1]q
  docker_compose_raw = %[3]q
  %[4]s
}
`, name, serverUUID, compose, extra)
}
