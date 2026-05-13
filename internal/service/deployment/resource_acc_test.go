package deployment_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestAccDeploymentResource_Create
// ---------------------------------------------------------------------------

func TestAccDeploymentResource_CreateImport(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-deploy")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_deployment.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_deployment.test", "status"),
				),
			},
			// Import
			{
				ResourceName:                         "coolify_deployment.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"triggers", "wait_for_completion"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["coolify_deployment.test"]
					return rs.Primary.Attributes["application_uuid"] + ":" + rs.Primary.Attributes["uuid"], nil
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccDeploymentDataSources
// ---------------------------------------------------------------------------

func TestAccDeploymentDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-deploy-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(name, serverUUID) + `
data "coolify_deployments" "all" {
  depends_on = [coolify_deployment.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_deployments.all", "deployments.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccDeploymentConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}

resource "coolify_deployment" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  triggers = {
    version = "1"
  }
}
`, name, serverUUID)
}
