package server_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-srv")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_server", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccServerConfig(name, privKey, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_server.test", "name", name),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "192.0.2.1"),
					resource.TestCheckResourceAttrSet("coolify_server.test", "private_key_uuid"),
				),
			},
			// Step 2: Update name and description
			{
				Config: testAccServerConfig(name+"-updated", privKey, `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_server.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_server.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_server.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"private_key_uuid"},
			},
		},
	})
}

func testAccServerConfig(name, privKey, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = "%[1]s-key"
  private_key = %[3]q
}

resource "coolify_server" "test" {
  name             = %[1]q
  ip               = "192.0.2.1"
  private_key_uuid = coolify_private_key.test.uuid
  %[2]s
}
`, name, extra, privKey)
}
