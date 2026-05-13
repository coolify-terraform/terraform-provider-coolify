package mysql_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
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
		CheckDestroy:             acctest.AccCheckDestroy("coolify_mysql_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.AccTestDatabaseConfig("coolify_mysql_database", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mysql_database.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_mysql_database.test", "name", name),
				),
			},
			// Step 2: Update description
			{
				Config: acctest.AccTestDatabaseConfig("coolify_mysql_database", name, serverUUID, `description = "updated mysql"`),
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
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_mysql_database.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"mysql_password", "mysql_root_password"},
			},
		},
	})
}
