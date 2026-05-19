package hetzner_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHetznerServerResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoHetznerToken(t)

	name := acctest.RandomWithPrefix("tf-acc-hetzner")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccHetznerServerConfig(name, privKey, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server_hetzner.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_server_hetzner.test", "ip"),
				),
			},
			// Update name.
			{
				Config: testAccHetznerServerConfig(name+"-renamed", privKey, ""),
				Check:  resource.TestCheckResourceAttr("coolify_server_hetzner.test", "name", name+"-renamed"),
			},
			// Import.
			{
				ResourceName:                         "coolify_server_hetzner.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_server_hetzner.test", "uuid"),
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid",
					"server_type",
					"location",
					"image",
					"hetzner_ssh_key_ids",
					"cloud_init_script",
					"instant_validate",
					"enable_ipv4",
					"enable_ipv6",
					"private_key_uuid",
				},
			},
		},
	})
}

func testAccHetznerServerConfig(name, privKey, extra string) string {
	return acctest.ConfigProviderBlock() + testAccHetznerCloudTokenConfig(name) + fmt.Sprintf(`
resource "coolify_private_key" "hetzner_test" {
  name        = "%[1]s-key"
  private_key = %[2]q
}

resource "coolify_server_hetzner" "test" {
  name                       = %[1]q
  cloud_provider_token_uuid  = coolify_cloud_token.hetzner_test.uuid
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = coolify_private_key.hetzner_test.uuid
  %[3]s
}
`, name, privKey, extra)
}
