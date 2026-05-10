package githubapp_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
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
				ImportStateVerifyIgnore:              []string{"client_secret", "private_key"},
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
  private_key     = <<-EOT
-----BEGIN RSA PRIVATE KEY-----
MIIBogIBAAJBALRiMLH00a6VK6aBIOxSjDQ3cBcVSaDXGfhLzJRNFp+t4/AOeDmR
5jXIx7DKXAIF9LRPz2gMjTb4i/r20hEh4cUCAwEAAQJBAJmHpJzk0fzYIYv3ihEE
3Ni7SIsMFCEzW0MREqYoLfpyBenGChQVqBqy9XAEiTHDhVsMb0ygDVRGGBGk0nkC
IQDjBCIEaeWV//pZGeJBU6o3JRxJV0rYpAf+0JCuXxvhfwIhAMuaVUIzMgEeClUH
7MYhb91EjG7RQcU0fYq+mUKPXaQfAiAXgdSDZvGhRHrFHGMLCcGI0EdCxKNcUmOb
sDijzrCVlQIgbbsHtPPG0oFkkRe8Y+FRZFyJBLlaCRxNyOWLzRYW/BsCIBFJ2Pla
0EQ2/JWFj1fOfsMnVMxOa2A1SL4lXEm6iNgV
-----END RSA PRIVATE KEY-----
EOT
}
`, name)
}
