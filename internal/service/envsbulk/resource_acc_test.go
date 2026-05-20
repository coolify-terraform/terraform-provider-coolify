package envsbulk_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEnvsBulk_Application(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-envsbulk")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccEnvsBulkConfig(name, serverUUID, `{ TEST_KEY = "test_value" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "resource_type", "application"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.TEST_KEY", "test_value"),
				),
			},
			{
				Config: testAccEnvsBulkConfig(name, serverUUID, `{ TEST_KEY = "updated_value", ANOTHER = "val" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.TEST_KEY", "updated_value"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.ANOTHER", "val"),
				),
			},
			{
				ResourceName:                         "coolify_envs_bulk.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccEnvsBulkImportStateIDFunc("coolify_application_dockerfile.test", "coolify_envs_bulk.test"),
				ImportStateVerify:                    false,
				ImportStateVerifyIdentifierAttribute: "resource_uuid",
			},
		},
	})
}

func testAccEnvsBulkConfig(name, serverUUID, vars string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %[1]q }
resource "coolify_application_dockerfile" "test" {
  project_uuid    = coolify_project.test.uuid
  server_uuid     = %[2]q
  name            = %[1]q
  dockerfile_location = base64encode("FROM nginx:alpine\nEXPOSE 80\n")
  ports_exposes   = "80"
}
resource "coolify_envs_bulk" "test" {
  resource_type = "application"
  resource_uuid = coolify_application_dockerfile.test.uuid
  variables     = %[3]s
}
`, name, serverUUID, vars)
}

func testAccEnvsBulkImportStateIDFunc(appResourceName, envsBulkResourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		appRS, ok := s.RootModule().Resources[appResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", appResourceName)
		}
		if _, ok := s.RootModule().Resources[envsBulkResourceName]; !ok {
			return "", fmt.Errorf("resource %s not found in state", envsBulkResourceName)
		}
		return "application/" + appRS.Primary.Attributes["uuid"], nil
	}
}
