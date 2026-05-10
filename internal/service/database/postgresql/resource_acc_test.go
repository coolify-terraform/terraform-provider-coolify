package postgresql_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPostgresqlDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-pg")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccPostgresqlConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_postgresql_database.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_postgresql_database.test", "image"),
				),
			},
			// Update description
			{
				Config: testAccPostgresqlConfig(name, serverUUID, `description = "Updated via acc test"`),
				Check: resource.TestCheckResourceAttr("coolify_postgresql_database.test", "description", "Updated via acc test"),
			},
			// Import
			{
				ResourceName:                         "coolify_postgresql_database.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"postgres_password"},
			},
		},
	})
}

func TestAccPostgresqlDatabaseDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-pg-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlConfig(name, serverUUID, "") + `
data "coolify_database" "test" {
  uuid = coolify_postgresql_database.test.uuid
}

data "coolify_databases" "all" {
  depends_on = [coolify_postgresql_database.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Single data source
					resource.TestCheckResourceAttrPair(
						"data.coolify_database.test", "uuid",
						"coolify_postgresql_database.test", "uuid",
					),
					resource.TestCheckResourceAttrPair(
						"data.coolify_database.test", "name",
						"coolify_postgresql_database.test", "name",
					),
					// List data source
					resource.TestCheckResourceAttrSet("data.coolify_databases.all", "databases.#"),
				),
			},
		},
	})
}

func testAccPostgresqlConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_postgresql_database" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
  %[3]s
}
`, name, serverUUID, extra)
}
