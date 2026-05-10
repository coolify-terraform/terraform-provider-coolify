package keydb_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKeydbDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-keydb")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccKeydbConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_keydb_database.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_keydb_database.test", "image"),
				),
			},
			// Step 2: Update description
			{
				Config: testAccKeydbConfig(name, serverUUID, `description = "Updated via acc test"`),
				Check: resource.TestCheckResourceAttr("coolify_keydb_database.test", "description", "Updated via acc test"),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_keydb_database.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func testAccKeydbConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_keydb_database" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
  %[3]s
}
`, name, serverUUID, extra)
}
