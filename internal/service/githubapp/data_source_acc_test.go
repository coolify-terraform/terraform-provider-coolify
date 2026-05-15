package githubapp_test

import (
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGitHubAppRepositoriesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp-repos")
	privateKeyName := acctest.RandomWithPrefix("tf-acc-ghapp-repos-key")
	privateKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccGitHubAppReposConfig(name, privateKeyName, privateKey),
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
	privateKeyName := acctest.RandomWithPrefix("tf-acc-ghapp-br-key")
	privateKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccGitHubAppBranchesConfig(name, privateKeyName, privateKey),
				ExpectError: regexp.MustCompile(`.+`),
			},
		},
	})
}

func testAccGitHubAppReposConfig(name, privateKeyName, privateKey string) string {
	return testAccGitHubAppConfig(name, privateKeyName, privateKey) + `
data "coolify_github_app_repositories" "test" {
  github_app_id = coolify_github_app.test.id
}
`
}

func testAccGitHubAppBranchesConfig(name, privateKeyName, privateKey string) string {
	return testAccGitHubAppConfig(name, privateKeyName, privateKey) + `
data "coolify_github_app_branches" "test" {
  github_app_id = coolify_github_app.test.id
  owner         = "coollabsio"
  repo          = "coolify"
}
`
}
