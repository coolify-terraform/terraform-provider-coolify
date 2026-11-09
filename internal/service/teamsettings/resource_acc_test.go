package teamsettings_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamSettingsResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.7.0")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_team_settings" "test" {
  scan_images_on_new_servers = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_team_settings.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_team_settings.test", "scan_images_on_new_servers", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_team_settings" "test" {
  scan_images_on_new_servers = false
}
`,
				Check: resource.TestCheckResourceAttr("coolify_team_settings.test", "scan_images_on_new_servers", "false"),
			},
			{
				ResourceName:      "coolify_team_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "current",
			},
		},
	})
}
