package vultr_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func skipIfNoVultrToken(t *testing.T) {
	t.Helper()
	if os.Getenv("COOLIFY_VULTR_TOKEN") == "" {
		t.Skip("COOLIFY_VULTR_TOKEN not set, skipping (Coolify proxies requests to Vultr API)")
	}
}

func testAccVultrCloudTokenConfig(name string) string {
	return `
resource "coolify_cloud_token" "vultr_test" {
  name           = "` + name + `"
  cloud_provider = "vultr"
  token          = "` + os.Getenv("COOLIFY_VULTR_TOKEN") + `"
}
`
}

func TestAccVultrServerResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoVultrToken(t)

	name := acctest.RandomWithPrefix("tf-acc-vultr")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_server_vultr", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: testAccVultrServerConfig(name, privKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server_vultr.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "name", name),
				),
			},
			{
				Config:             testAccVultrServerConfig(name, privKey),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccVultrServerConfig(name+"-renamed", privKey),
				Check:  resource.TestCheckResourceAttr("coolify_server_vultr.test", "name", name+"-renamed"),
			},
			{
				ResourceName:                         "coolify_server_vultr.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_server_vultr.test", "uuid"),
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid", "region", "plan", "os_id",
					"vultr_ssh_key_ids", "cloud_init_script", "instant_validate",
					"enable_ipv6", "disable_public_ipv4", "private_key_uuid",
				},
			},
		},
	})
}

func testAccVultrServerConfig(name, privKey string) string {
	return acctest.ConfigProviderBlock() + testAccVultrCloudTokenConfig(name) + fmt.Sprintf(`
resource "coolify_private_key" "vultr_test" {
  name        = "%[1]s-key"
  private_key = %[2]q
}

resource "coolify_server_vultr" "test" {
  name                      = %[1]q
  cloud_provider_token_uuid = coolify_cloud_token.vultr_test.uuid
  region                    = "ewr"
  plan                      = "vc2-1c-1gb"
  os_id                     = 1743
  private_key_uuid          = coolify_private_key.vultr_test.uuid
}
`, name, privKey)
}
