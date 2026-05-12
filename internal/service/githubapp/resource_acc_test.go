package githubapp_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGitHubAppResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccGitHubAppConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "name", name),
					resource.TestCheckResourceAttr("coolify_github_app.test", "app_id", "12345"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "installation_id", "67890"),
					resource.TestCheckResourceAttr("coolify_github_app.test", "client_id", "Iv1.fake123456789"),
				),
			},
			// Step 2: Import by ID
			{
				ResourceName:                         "coolify_github_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"client_secret", "private_key_uuid"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_github_app.test"]
					if !ok {
						return "", fmt.Errorf("resource coolify_github_app.test not found")
					}
					return rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func TestAccGitHubAppDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGitHubAppConfig(name) + `
data "coolify_github_apps" "all" {
  depends_on = [coolify_github_app.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app.test", "id"),
					resource.TestMatchResourceAttr("data.coolify_github_apps.all", "github_apps.#", regexp.MustCompile(`[1-9]\d*`)),
				),
			},
		},
	})
}

func testAccGitHubAppConfig(name string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_github_app" "test" {
  name            = %[1]q
  app_id          = 12345
  installation_id = 67890
  client_id       = "Iv1.fake123456789"
  client_secret   = "fake-client-secret-value"
  private_key_uuid = "pk-uuid-acctest"
}
`, name)
}
