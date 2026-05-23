package mariadb_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMariadbDatabaseDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-maria-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_mariadb", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_mariadb", name, serverUUID, "") + `
data "coolify_database" "test" {
  uuid = coolify_database_mariadb.test.uuid
}

data "coolify_databases" "all" {
  depends_on = [coolify_database_mariadb.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "uuid", "coolify_database_mariadb.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "name", "coolify_database_mariadb.test", "name"),
					resource.TestCheckResourceAttrSet("data.coolify_databases.all", "databases.#"),
				),
			},
		},
	})
}

func TestAccMariadbDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-mariadb")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_mariadb", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_mariadb", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_mariadb.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_mariadb.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_database_mariadb.test", "image"),
				),
			},
			// Idempotency check
			{
				Config:             acctest.AccTestDatabaseConfig("coolify_database_mariadb", name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_mariadb", name, serverUUID, `description = "Updated via acc test"`),
				Check:  resource.TestCheckResourceAttr("coolify_database_mariadb.test", "description", "Updated via acc test"),
			},
			{
				ResourceName:                         "coolify_database_mariadb.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_database_mariadb.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"mariadb_password", "mariadb_root_password", "project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}
