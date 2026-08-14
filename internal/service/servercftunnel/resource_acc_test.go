package servercftunnel_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerCFTunnelResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_cloudflare_tunnel" "test" {
  server_uuid          = %q
  is_cloudflare_tunnel = false
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_cloudflare_tunnel.test", "is_cloudflare_tunnel", "false"),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_cloudflare_tunnel" "test" {
  server_uuid          = %q
  is_cloudflare_tunnel = false
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_cloudflare_tunnel.test", "server_uuid", serverUUID),
			},
			{
				ResourceName:                         "coolify_server_cloudflare_tunnel.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}
