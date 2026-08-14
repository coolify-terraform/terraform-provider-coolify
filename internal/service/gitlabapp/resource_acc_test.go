package gitlabapp_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGitLabAppResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	name := acctest.RandomWithPrefix("tf-acc-glapp")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_gitlab_app" "test" {
  name     = %q
  html_url = "https://gitlab.example.com"
}
data "coolify_gitlab_app" "by_id" { id = coolify_gitlab_app.test.id }
data "coolify_gitlab_apps" "all" {}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_gitlab_app.test", "id"),
					resource.TestCheckResourceAttrPair("data.coolify_gitlab_app.by_id", "name", "coolify_gitlab_app.test", "name"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_gitlab_app" "test" {
  name     = %q
  html_url = "https://gitlab.example.com"
}
`, name+"-upd"),
				Check: resource.TestCheckResourceAttr("coolify_gitlab_app.test", "name", name+"-upd"),
			},
			{
				ResourceName:                         "coolify_gitlab_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"client_secret", "webhook_token"},
			},
		},
	})
}
