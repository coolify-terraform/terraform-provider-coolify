package keydb_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
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
		CheckDestroy:             acctest.AccCheckDestroy("coolify_keydb_database", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.AccTestDatabaseConfig("coolify_keydb_database", name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_keydb_database.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_keydb_database.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_keydb_database.test", "image"),
				),
			},
			// Step 2: Update description
			{
				Config: acctest.AccTestDatabaseConfig("coolify_keydb_database", name, serverUUID, `description = "Updated via acc test"`),
				Check:  resource.TestCheckResourceAttr("coolify_keydb_database.test", "description", "Updated via acc test"),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_keydb_database.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_keydb_database.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}
