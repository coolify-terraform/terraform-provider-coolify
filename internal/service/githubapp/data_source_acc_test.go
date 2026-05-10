package githubapp_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGitHubAppRepositoriesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp-repos")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccGitHubAppReposConfig(name),
				ExpectError: regexp.MustCompile(`.+`),
			},
		},
	})
}

func TestAccGitHubAppBranchesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp-br")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccGitHubAppBranchesConfig(name),
				ExpectError: regexp.MustCompile(`.+`),
			},
		},
	})
}

func testAccGitHubAppReposConfig(name string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_github_app" "test" {
  name            = %[1]q
  app_id          = 12345
  installation_id = 67890
  client_id       = "Iv1.dummy"
  client_secret   = "dummysecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\ndummy\n-----END RSA PRIVATE KEY-----"
}

data "coolify_github_app_repositories" "test" {
  github_app_id = coolify_github_app.test.id
}
`, name)
}

func testAccGitHubAppBranchesConfig(name string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_github_app" "test" {
  name            = %[1]q
  app_id          = 12345
  installation_id = 67890
  client_id       = "Iv1.dummy"
  client_secret   = "dummysecret"
  private_key     = "-----BEGIN RSA PRIVATE KEY-----\ndummy\n-----END RSA PRIVATE KEY-----"
}

data "coolify_github_app_branches" "test" {
  github_app_id = coolify_github_app.test.id
  owner         = "coollabsio"
  repo          = "coolify"
}
`, name)
}
