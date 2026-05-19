package resourceaction_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceActionResource_StartDatabase(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-action")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceActionConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_resource_action.start_db", "resource_uuid"),
					resource.TestCheckResourceAttr("coolify_resource_action.start_db", "resource_type", "database"),
					resource.TestCheckResourceAttr("coolify_resource_action.start_db", "action", "start"),
				),
			},
		},
	})
}

func testAccResourceActionConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_database_postgresql" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  environment_name = "production"
}

resource "coolify_resource_action" "start_db" {
  resource_uuid = coolify_database_postgresql.test.uuid
  resource_type = "database"
  action        = "start"
}
`, name, serverUUID)
}
