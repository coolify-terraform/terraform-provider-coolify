package serverproxy_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerProxyResource_CRUD(t *testing.T) {
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
resource "coolify_server_proxy" "test" {
  server_uuid = %q
  proxy_type  = "traefik"
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_proxy.test", "server_uuid", serverUUID),
			},
			{
				// Do not set redirect_enabled = false. Coolify's PATCH uses
				// $request->has('redirect_enabled'); Laravel has() is false
				// for JSON false, so the write is ignored and GET stays true
				// (default). Same class as empty domains (#647).
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_proxy" "test" {
  server_uuid  = %q
  proxy_type   = "traefik"
  redirect_url = "https://example.invalid"
}
`, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "proxy_type", "traefik"),
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "redirect_url", "https://example.invalid"),
				),
			},
			{
				ResourceName:                         "coolify_server_proxy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}
