package cloudtokenvalidate_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudTokenValidate_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	token := os.Getenv("COOLIFY_HETZNER_TOKEN")
	if token == "" {
		t.Skip("COOLIFY_HETZNER_TOKEN not set, skipping (Coolify validates token against Hetzner API)")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudTokenValidateConfig(token),
				Check:  resource.TestCheckResourceAttrSet("coolify_cloud_token_validate.test", "cloud_token_uuid"),
			},
		},
	})
}

func testAccCloudTokenValidateConfig(token string) string {
	name := acctest.RandomWithPrefix("tf-acc-validate")
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_token" "test" {
  name           = %q
  cloud_provider = "hetzner"
  token          = %q
}
resource "coolify_cloud_token_validate" "test" {
  cloud_token_uuid = coolify_cloud_token.test.uuid
}
`, name, token)
}
