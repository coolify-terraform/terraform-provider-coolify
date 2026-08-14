package servercftunnel_test

import (
	"fmt"
	"testing"

	"regexp"

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
				// CI Coolify uses the localhost server, which rejects tunnel config.
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_cloudflare_tunnel" "test" {
  server_uuid          = %q
  is_cloudflare_tunnel = true
}
`, serverUUID),
				ExpectError: regexp.MustCompile(`localhost server|Cloudflare Tunnel`),
			},
		},
	})
}
