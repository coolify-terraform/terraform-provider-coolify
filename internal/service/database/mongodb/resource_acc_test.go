package mongodb_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMongodbDatabaseDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-mongo-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_mongodb_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_mongodb_database", name, serverUUID, "") + `
data "coolify_database" "test" {
  uuid = coolify_mongodb_database.test.uuid
}

data "coolify_databases" "all" {
  depends_on = [coolify_mongodb_database.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "uuid", "coolify_mongodb_database.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_database.test", "name", "coolify_mongodb_database.test", "name"),
					resource.TestCheckResourceAttrSet("data.coolify_databases.all", "databases.#"),
				),
			},
		},
	})
}

func TestAccMongodbDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-mongodb")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_mongodb_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_mongodb_database", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_mongodb_database.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_mongodb_database.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_mongodb_database.test", "image"),
				),
			},
			// Idempotency check
			{
				Config:             acctest.AccTestDatabaseConfig("coolify_mongodb_database", name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: acctest.AccTestDatabaseConfig("coolify_mongodb_database", name, serverUUID, `description = "Updated via acc test"`),
				Check:  resource.TestCheckResourceAttr("coolify_mongodb_database.test", "description", "Updated via acc test"),
			},
			{
				ResourceName:                         "coolify_mongodb_database.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_mongodb_database.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"mongo_initdb_root_password", "project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}
