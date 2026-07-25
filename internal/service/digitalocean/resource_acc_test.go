package digitalocean_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func skipIfNoDigitalOceanToken(t *testing.T) {
	t.Helper()
	if os.Getenv("COOLIFY_DIGITALOCEAN_TOKEN") == "" {
		t.Skip("COOLIFY_DIGITALOCEAN_TOKEN not set, skipping (Coolify proxies requests to DigitalOcean API)")
	}
}

func testAccDigitalOceanCloudTokenConfig(name string) string {
	return `
resource "coolify_cloud_token" "do_test" {
  name           = "` + name + `"
  cloud_provider = "digitalocean"
  token          = "` + os.Getenv("COOLIFY_DIGITALOCEAN_TOKEN") + `"
}
`
}

func TestAccDigitalOceanServerResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoDigitalOceanToken(t)

	name := acctest.RandomWithPrefix("tf-acc-do")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_server_digitalocean", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: testAccDigitalOceanServerConfig(name, privKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server_digitalocean.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "name", name),
				),
			},
			{
				Config:             testAccDigitalOceanServerConfig(name, privKey),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccDigitalOceanServerConfig(name+"-renamed", privKey),
				Check:  resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "name", name+"-renamed"),
			},
			{
				ResourceName:                         "coolify_server_digitalocean.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_server_digitalocean.test", "uuid"),
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid", "region", "size", "image",
					"digitalocean_ssh_key_ids", "cloud_init_script", "instant_validate",
					"enable_ipv6", "monitoring", "private_key_uuid",
				},
			},
		},
	})
}

func testAccDigitalOceanServerConfig(name, privKey string) string {
	return acctest.ConfigProviderBlock() + testAccDigitalOceanCloudTokenConfig(name) + fmt.Sprintf(`
resource "coolify_private_key" "do_test" {
  name        = "%[1]s-key"
  private_key = %[2]q
}

resource "coolify_server_digitalocean" "test" {
  name                      = %[1]q
  cloud_provider_token_uuid = coolify_cloud_token.do_test.uuid
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = coolify_private_key.do_test.uuid
}
`, name, privKey)
}
