package backup_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDatabaseBackupResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-bkp")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_database_backup", "database_uuid", "/api/v1/databases/%s/backups"),
		Steps: []resource.TestStep{
			// Create with initial frequency
			{
				Config: testAccBackupConfig(name, serverUUID, "0 2 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_backup.test", "id"),
					resource.TestCheckResourceAttrSet("coolify_database_backup.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_amount_locally", "7"),
				),
			},
			// Idempotency check
			{
				Config:             testAccBackupConfig(name, serverUUID, "0 2 * * *"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update frequency
			{
				Config: testAccBackupConfig(name, serverUUID, "0 4 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 4 * * *"),
				),
			},
			// Import
			{
				ResourceName:                         "coolify_database_backup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_database_backup.test"]
					if !ok {
						return "", fmt.Errorf("resource coolify_database_backup.test not found in state")
					}
					return rs.Primary.Attributes["database_uuid"] + ":" + rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func testAccBackupConfig(name, serverUUID, frequency string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_database_postgresql" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
}

resource "coolify_database_backup" "test" {
  database_uuid = coolify_database_postgresql.test.uuid
  frequency     = %[3]q
  enabled       = true
  retain_amount_locally   = 7
}
`, name, serverUUID, frequency)
}

// ---------------------------------------------------------------------------
// TestAccDatabaseBackupResource_S3 — S3 backup workflow against real MinIO
// ---------------------------------------------------------------------------

func TestAccDatabaseBackupResource_S3(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	s3UUID := acctest.AccTestS3StorageUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-s3bkp")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_database_backup", "database_uuid", "/api/v1/databases/%s/backups"),
		Steps: []resource.TestStep{
			// Create with S3 enabled
			{
				Config: testAccBackupS3Config(name, serverUUID, s3UUID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_backup.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "save_s3", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "s3_storage_uuid", s3UUID),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 3 * * *"),
				),
			},
			// Idempotency: no plan diff on re-apply
			{
				Config:             testAccBackupS3Config(name, serverUUID, s3UUID, true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Import
			{
				ResourceName:                         "coolify_database_backup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_database_backup.test"]
					if !ok {
						return "", fmt.Errorf("resource coolify_database_backup.test not found in state")
					}
					return rs.Primary.Attributes["database_uuid"] + ":" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{
					"s3_storage_uuid",
					"retain_amount_locally", "retain_days_locally", "retain_max_storage_locally",
					"retain_amount_s3", "retain_days_s3", "retain_max_storage_s3",
					"timeout", "databases_to_backup",
				},
			},
		},
	})
}

func testAccBackupS3Config(name, serverUUID, s3UUID string, saveS3 bool) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_database_postgresql" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
}

resource "coolify_database_backup" "test" {
  database_uuid   = coolify_database_postgresql.test.uuid
  frequency       = "0 3 * * *"
  enabled         = true
  save_s3         = %[4]t
  s3_storage_uuid = %[3]q
}
`, name, serverUUID, s3UUID, saveS3)
}

// ---------------------------------------------------------------------------
// TestAccBackupExecutionsDataSource
// ---------------------------------------------------------------------------

func TestAccBackupExecutionsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-bkpexec")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_database_backup", "database_uuid", "/api/v1/databases/%s/backups"),
		Steps: []resource.TestStep{
			{
				Config: testAccBackupExecutionsConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_backup_executions.test", "executions.#"),
				),
			},
		},
	})
}

func testAccBackupExecutionsConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}
resource "coolify_database_postgresql" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
}
resource "coolify_database_backup" "test" {
  database_uuid = coolify_database_postgresql.test.uuid
  frequency     = "0 2 * * *"
  enabled       = true
}
data "coolify_backup_executions" "test" {
  database_uuid = coolify_database_postgresql.test.uuid
  backup_uuid   = coolify_database_backup.test.uuid
}
`, name, serverUUID)
}
