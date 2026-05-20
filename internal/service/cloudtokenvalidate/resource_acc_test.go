package cloudtokenvalidate_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudTokenValidate_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	// This test requires a valid cloud token UUID. Set COOLIFY_CLOUD_TOKEN_UUID
	// to a Hetzner token UUID registered in Coolify.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudTokenValidateConfig(),
				Check:  resource.TestCheckResourceAttrSet("coolify_cloud_token_validate.test", "cloud_token_uuid"),
			},
		},
	})
}

func testAccCloudTokenValidateConfig() string {
	name := acctest.RandomWithPrefix("tf-acc-validate")
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_token" "test" {
  name  = %q
  token = "test-token-for-validation"
}
resource "coolify_cloud_token_validate" "test" {
  cloud_token_uuid = coolify_cloud_token.test.uuid
}
`, name)
}
