package instanceemail_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccInstanceEmailSettingsDataSource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	accSkipIfNoInstanceEmailAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
data "coolify_instance_email_settings" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "id", "current"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.coolify_instance_email_settings.test"]
						if !ok {
							return fmt.Errorf("data.coolify_instance_email_settings.test not in state")
						}
						v := rs.Primary.Attributes["smtp_enabled"]
						if v != "true" && v != "false" {
							return fmt.Errorf("smtp_enabled = %q, want true or false", v)
						}
						return nil
					},
				),
			},
		},
	})
}
