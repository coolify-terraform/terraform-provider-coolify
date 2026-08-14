package serversentinel_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerSentinelResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.1.1")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_sentinel" "test" {
  server_uuid         = %q
  is_sentinel_enabled = false
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_sentinel_enabled", "false"),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_sentinel" "test" {
  server_uuid          = %q
  is_sentinel_enabled  = false
  is_metrics_enabled   = false
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_sentinel.test", "is_metrics_enabled", "false"),
			},
			{
				ResourceName:                         "coolify_server_sentinel.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}
