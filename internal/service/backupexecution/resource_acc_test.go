package backupexecution_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBackupExecution_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-backupexec")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBackupExecutionConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_backup_execution.test", "database_uuid"),
					resource.TestCheckResourceAttrSet("coolify_backup_execution.test", "backup_uuid"),
					resource.TestCheckResourceAttrSet("coolify_backup_execution.test", "execution_uuid"),
				),
			},
			{
				ResourceName:                         "coolify_backup_execution.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "execution_uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["coolify_backup_execution.test"]
					if rs == nil {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["database_uuid"] + ":" +
						rs.Primary.Attributes["backup_uuid"] + ":" +
						rs.Primary.Attributes["execution_uuid"], nil
				},
			},
		},
	})
}

func testAccBackupExecutionConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %[1]q }
resource "coolify_database_postgresql" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
}
resource "coolify_database_backup" "test" {
  database_uuid = coolify_database_postgresql.test.uuid
  frequency     = "0 3 * * *"
  enabled       = true
}
resource "coolify_backup_execution" "test" {
  database_uuid  = coolify_database_postgresql.test.uuid
  backup_uuid    = coolify_database_backup.test.uuid
  execution_uuid = "550e8400-e29b-41d4-a716-446655440099"
}
`, name, serverUUID)
}
