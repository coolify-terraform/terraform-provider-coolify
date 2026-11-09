package serverimagescan_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerImageScanningResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.7.0")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_image_scanning" "test" {
  server_uuid           = %q
  is_image_scan_enabled = true
}
`, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "server_uuid", serverUUID),
					resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "is_image_scan_enabled", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_image_scanning" "test" {
  server_uuid           = %q
  is_image_scan_enabled = false
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "is_image_scan_enabled", "false"),
			},
			{
				ResourceName:                         "coolify_server_image_scanning.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
				ImportStateVerifyIgnore: []string{
					"scanner_state",
					"scanner_failed_step",
					"scanner_built_at",
					"scanner_disk_used_bytes",
				},
			},
		},
	})
}
