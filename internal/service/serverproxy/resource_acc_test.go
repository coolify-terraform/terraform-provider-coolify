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
				// redirect_url uses exists() and persists, but SafeExternalUrl
				// requires a resolvable host (example.invalid 422s).
				// redirect_enabled = false is stored. Coolify >= 4.3.0 uses
				// Laravel 12, and has() sees a JSON false key.
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_proxy" "test" {
  server_uuid      = %q
  proxy_type       = "traefik"
  redirect_url     = "https://example.com"
  redirect_enabled = false
}
`, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "proxy_type", "traefik"),
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "redirect_url", "https://example.com"),
					resource.TestCheckResourceAttr("coolify_server_proxy.test", "redirect_enabled", "false"),
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
