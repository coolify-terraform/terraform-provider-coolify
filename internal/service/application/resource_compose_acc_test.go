package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
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
		CheckDestroy:             acctest.AccCheckDestroy("coolify_docker_compose_application", "/api/v1/applications/"),
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
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_docker_compose_application.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

func testAccDockerComposeAppConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_docker_compose_application" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
  docker_compose_raw = base64encode(<<-YAML
    services:
      web:
        image: nginx:alpine
        ports:
          - "80:80"
  YAML
  )
  %[3]s
}
`, name, serverUUID, extra)
}
