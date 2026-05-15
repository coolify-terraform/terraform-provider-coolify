package clickhouse_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClickhouseDatabaseDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ch-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_clickhouse_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_clickhouse_database", name, serverUUID, "") + `
data "coolify_database" "test" {
  uuid = coolify_clickhouse_database.test.uuid
}

data "coolify_databases" "all" {
  depends_on = [coolify_clickhouse_database.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "uuid", "coolify_clickhouse_database.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "name", "coolify_clickhouse_database.test", "name"),
					resource.TestCheckResourceAttrSet("data.coolify_databases.all", "databases.#"),
				),
			},
		},
	})
}

func TestAccClickhouseDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-clickhouse")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_clickhouse_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_clickhouse_database", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_clickhouse_database.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_clickhouse_database.test", "image"),
				),
			},
			// Idempotency check
			{
				Config:             acctest.AccTestDatabaseConfig("coolify_clickhouse_database", name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: acctest.AccTestDatabaseConfig("coolify_clickhouse_database", name, serverUUID, `description = "Updated via acc test"`),
				Check:  resource.TestCheckResourceAttr("coolify_clickhouse_database.test", "description", "Updated via acc test"),
			},
			{
				ResourceName:                         "coolify_clickhouse_database.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_clickhouse_database.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"clickhouse_admin_password", "project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}
