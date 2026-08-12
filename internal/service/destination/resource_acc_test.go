package destination_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	// DestinationsController create: Coolify >= v4.2.0 (POST .../servers/{uuid}/destinations).
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")

	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dest")
	network := acctest.RandomWithPrefix("tfnet")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_destination", "/api/v1/destinations/"),
		Steps: []resource.TestStep{
			{
				Config: testAccDestinationConfig(serverUUID, name, network),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_destination.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_destination.test", "network", network),
					resource.TestCheckResourceAttr("coolify_destination.test", "name", name),
					resource.TestCheckResourceAttr("coolify_destination.test", "type", "standalone"),
				),
			},
			{
				Config:             testAccDestinationConfig(serverUUID, name, network),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:                         "coolify_destination.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_destination.test", "uuid"),
			},
		},
	})
}

func testAccDestinationConfig(serverUUID, name, network string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_destination" "test" {
  server_uuid = %q
  network     = %q
  name        = %q
  type        = "standalone"
}
`, serverUUID, network, name)
}
