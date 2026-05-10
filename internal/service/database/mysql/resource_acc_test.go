package mysql_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMysqlDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-mysql")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccMysqlConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mysql_database.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "name", name),
				),
			},
			// Step 2: Update description
			{
				Config: testAccMysqlConfig(name, serverUUID, `description = "updated mysql"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mysql_database.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "name", name),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "description", "updated mysql"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_mysql_database.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"mysql_password", "mysql_root_password"},
			},
		},
	})
}

func testAccMysqlConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %[1]q }
resource "coolify_mysql_database" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
  %[3]s
}
`, name, serverUUID, extra)
}
