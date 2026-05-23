package servervalidate_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerValidate_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccServerValidateConfig(serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_validate.test", "server_uuid", serverUUID),
					resource.TestCheckResourceAttrSet("coolify_server_validate.test", "valid"),
				),
			},
		},
	})
}

func testAccServerValidateConfig(serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_validate" "test" {
  server_uuid = %q
}
`, serverUUID)
}
