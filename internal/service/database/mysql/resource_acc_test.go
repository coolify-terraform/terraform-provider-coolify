package mysql_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMysqlDatabaseDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-mysql-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_mysql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_mysql", name, serverUUID, "") + `
data "coolify_database" "test" {
  uuid = coolify_database_mysql.test.uuid
}

data "coolify_databases" "all" {
  depends_on = [coolify_database_mysql.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "uuid", "coolify_database_mysql.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "name", "coolify_database_mysql.test", "name"),
					resource.TestCheckResourceAttrSet("data.coolify_databases.all", "databases.#"),
				),
			},
		},
	})
}

func TestAccMysqlDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-mysql")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_mysql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_mysql", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_mysql.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_mysql.test", "name", name),
				),
			},
			// Idempotency check
			{
				Config:             acctest.AccTestDatabaseConfig("coolify_database_mysql", name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update description
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_mysql", name, serverUUID, `description = "updated mysql"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_mysql.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_mysql.test", "name", name),
					resource.TestCheckResourceAttr("coolify_database_mysql.test", "description", "updated mysql"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_database_mysql.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_database_mysql.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"mysql_password", "mysql_root_password", "project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}
