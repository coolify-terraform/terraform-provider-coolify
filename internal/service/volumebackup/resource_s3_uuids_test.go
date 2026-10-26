package volumebackup_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestStorageBackupResource_S3StorageUUIDs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		send    bool
	}{
		{version: "4.5.0", send: true},
		{version: "4.4.2", send: false},
		{version: "", send: true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			label := tc.version
			if label == "" {
				label = "empty"
			}
			mock := newVolumeBackupMockVersion(t, tc.version, false)
			defer mock.Close()
			list := `["aaaa1111-2222-3333-4444-555566667777","bbbb1111-2222-3333-4444-555566667777"]`
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				CheckDestroy:             mock.checkDestroy(),
				Steps: []resource.TestStep{
					{
						Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
  enabled          = true
  save_s3          = true
  s3_storage_uuids = ` + list + `
}
`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_storage_backup.test", "s3_storage_uuids.0", "aaaa1111-2222-3333-4444-555566667777"),
							func(*terraform.State) error {
								mock.mu.Lock()
								defer mock.mu.Unlock()
								got := len(mock.lastS3UUIDs) > 0
								if got != tc.send {
									return fmt.Errorf("Coolify %s sent s3_storage_uuids=%v, want sent=%v (%v)", label, mock.lastS3UUIDs, tc.send, mock.lastS3UUIDs)
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}
