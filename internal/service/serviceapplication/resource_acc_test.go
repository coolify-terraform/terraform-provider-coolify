package serviceapplication_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServiceApplicationResource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	// GET /services/{uuid}/applications exists from Coolify 4.2.0.
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-svcapp")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAppConfig(name, serverUUID, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_service_applications.test", "applications.0.uuid"),
					resource.TestCheckResourceAttr("coolify_service_application.test", "description", "first"),
				),
			},
			{
				Config: testAccServiceAppConfig(name, serverUUID, "second"),
				Check:  resource.TestCheckResourceAttr("coolify_service_application.test", "description", "second"),
			},
			{
				ResourceName:                         "coolify_service_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_service_application.test"]
					if !ok {
						return "", fmt.Errorf("missing coolify_service_application.test")
					}
					return rs.Primary.Attributes["service_uuid"] + "/" + rs.Primary.Attributes["uuid"], nil
				},
				ImportStateVerifyIgnore: []string{"force_domain_override"},
			},
		},
	})
}

func testAccServiceAppConfig(name, serverUUID, description string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}
resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  type         = "uptime-kuma"
  name         = %[1]q
}
data "coolify_service_applications" "test" {
  service_uuid = coolify_service.test.uuid
}
resource "coolify_service_application" "test" {
  service_uuid = coolify_service.test.uuid
  uuid         = data.coolify_service_applications.test.applications[0].uuid
  description  = %[3]q
}
`, name, serverUUID, description)
}
