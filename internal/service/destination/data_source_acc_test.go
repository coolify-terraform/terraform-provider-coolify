package destination_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dest-ds")
	network := acctest.RandomWithPrefix("tfnetds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testAccDestinationConfig(serverUUID, name, network) + `
data "coolify_destination" "test" {
  uuid = coolify_destination.test.uuid
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.coolify_destination.test", "uuid", "coolify_destination.test", "uuid"),
				resource.TestCheckResourceAttr("data.coolify_destination.test", "network", network),
			),
		}},
	})
}

func TestAccDestinationsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dests")
	network := acctest.RandomWithPrefix("tfnetlist")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testAccDestinationConfig(serverUUID, name, network) + fmt.Sprintf(`
data "coolify_destinations" "test" {
  filter {
    name   = "network"
    values = [%q]
  }
  depends_on = [coolify_destination.test]
}
`, network),
			Check: resource.TestCheckResourceAttrSet("data.coolify_destinations.test", "destinations.#"),
		}},
	})
}
