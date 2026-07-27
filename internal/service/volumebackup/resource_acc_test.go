package volumebackup_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccStorageBackupResource_CRUD exercises "coolify_storage_backup" against a real Coolify
// instance with VolumeBackupsController (Coolify v4.x after coollabsio/coolify#10946; not in tag v4.2.0).
func TestAccStorageBackupResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	// Needs Coolify v4.x including #10946 + a disposable storage UUID. Unit tests cover the resource.
	t.Skip("coolify_storage_backup acceptance needs Coolify v4.x with volume-backup routes (#10946+) and a test storage UUID")

	appUUID := "00000000-0000-4000-8000-000000000001"
	storUUID := "00000000-0000-4000-8000-000000000002"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBackupConfig(appUUID, storUUID, "0 2 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_storage_backup.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 2 * * *"),
				),
			},
			{
				Config: testAccStorageBackupConfig(appUUID, storUUID, "0 3 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 3 * * *"),
				),
			},
			{
				ResourceName:                         "coolify_storage_backup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "storage_uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return "application:" + appUUID + ":" + storUUID, nil
				},
				ImportStateVerifyIgnore: []string{
					"uuid", "frequency", "enabled", "save_s3", "disable_local_backup",
					"stop_during_backup", "s3_storage_uuid", "storage_type",
					"retention_amount_locally", "retention_days_locally", "retention_max_storage_locally",
					"retention_amount_s3", "retention_days_s3", "retention_max_storage_s3", "timeout",
				},
			},
		},
	})
}

func testAccStorageBackupConfig(appUUID, storUUID, frequency string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_storage_backup" "test" {
  application_uuid = %q
  storage_uuid     = %q
  frequency        = %q
}
`, appUUID, storUUID, frequency)
}
