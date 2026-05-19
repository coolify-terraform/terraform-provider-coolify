package postgresql_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
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
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_postgresql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_postgresql", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_postgresql.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_postgresql.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_database_postgresql.test", "image"),
				),
			},
			// Idempotency check
			{
				Config:             acctest.AccTestDatabaseConfig("coolify_database_postgresql", name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update description
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_postgresql", name, serverUUID, `description = "Updated via acc test"`),
				Check:  resource.TestCheckResourceAttr("coolify_database_postgresql.test", "description", "Updated via acc test"),
			},
			// Import
			{
				ResourceName:                         "coolify_database_postgresql.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_database_postgresql.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"postgres_password", "project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}

func TestAccPostgresqlDatabaseResource_Disappears(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-pg-dis")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_postgresql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_postgresql", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_postgresql.test", "uuid"),
					acctest.AccCheckResourceDisappears("coolify_database_postgresql.test", "/api/v1/databases/"),
				),
				ExpectNonEmptyPlan: true,
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
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_postgresql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_postgresql", name, serverUUID, "") + `
data "coolify_database" "test" {
  uuid = coolify_database_postgresql.test.uuid
}

data "coolify_databases" "all" {
  depends_on = [coolify_database_postgresql.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Single data source
					resource.TestCheckResourceAttrPair(
						"data.coolify_database.test", "uuid",
						"coolify_database_postgresql.test", "uuid",
					),
					resource.TestCheckResourceAttrPair(
						"data.coolify_database.test", "name",
						"coolify_database_postgresql.test", "name",
					),
					// List data source
					resource.TestCheckResourceAttrSet("data.coolify_databases.all", "databases.#"),
				),
			},
		},
	})
}
