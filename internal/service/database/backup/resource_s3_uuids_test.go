package backup_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDatabaseBackupResource_S3StorageUUIDs(t *testing.T) {
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
			version := tc.version
			if version == "" {
				version = "empty"
			}
			srv, state := newMockBackupServerVersion(tc.version)
			defer srv.Close()
			first := `["aaaa1111-2222-3333-4444-555566667777","bbbb1111-2222-3333-4444-555566667777"]`
			second := `["aaaa1111-2222-3333-4444-555566667777","cccc1111-2222-3333-4444-555566667777"]`
			checkSent := func(want []string) resource.TestCheckFunc {
				return func(*terraform.State) error {
					state.mu.Lock()
					defer state.mu.Unlock()
					if state.sawS3UUIDs != tc.send {
						return fmt.Errorf("Coolify %s saw s3_storage_uuids=%v, want %v", version, state.sawS3UUIDs, tc.send)
					}
					if !tc.send {
						return nil
					}
					if len(state.s3UUIDs) != len(want) {
						return fmt.Errorf("s3_storage_uuids=%v, want %v", state.s3UUIDs, want)
					}
					for i := range want {
						if state.s3UUIDs[i] != want[i] {
							return fmt.Errorf("s3_storage_uuids=%v, want %v", state.s3UUIDs, want)
						}
					}
					return nil
				}
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				CheckDestroy:             checkBackupDestroy(srv.URL),
				Steps: []resource.TestStep{
					{
						Config: testBackupConfig(srv.URL, `
							database_uuid    = "eeee0001-0001-4000-8000-000000000001"
							frequency        = "0 2 * * *"
							enabled          = true
							save_s3          = true
							s3_storage_uuids = `+first+`
						`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_database_backup.test", "s3_storage_uuids.0", "aaaa1111-2222-3333-4444-555566667777"),
							resource.TestCheckResourceAttr("coolify_database_backup.test", "s3_storage_uuids.1", "bbbb1111-2222-3333-4444-555566667777"),
							checkSent([]string{"aaaa1111-2222-3333-4444-555566667777", "bbbb1111-2222-3333-4444-555566667777"}),
						),
					},
					{
						Config: testBackupConfig(srv.URL, `
							database_uuid    = "eeee0001-0001-4000-8000-000000000001"
							frequency        = "0 2 * * *"
							enabled          = true
							save_s3          = true
							s3_storage_uuids = `+second+`
						`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_database_backup.test", "s3_storage_uuids.1", "cccc1111-2222-3333-4444-555566667777"),
							checkSent([]string{"aaaa1111-2222-3333-4444-555566667777", "cccc1111-2222-3333-4444-555566667777"}),
						),
					},
				},
			})
		})
	}
}
