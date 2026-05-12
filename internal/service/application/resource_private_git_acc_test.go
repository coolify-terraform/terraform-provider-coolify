package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateGitApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-privgit")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccPrivateGitAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_git_application.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "git_repository", "git@github.com:coollabsio/coolify-examples.git"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "ports_exposes", "3000"),
				),
			},
			// Step 2: Update description
			{
				Config: testAccPrivateGitAppConfig(name, serverUUID, `description = "Updated private git app"`),
				Check:  resource.TestCheckResourceAttr("coolify_private_git_application.test", "description", "Updated private git app"),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_private_git_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name", "private_key_uuid"},
			},
		},
	})
}

func testAccPrivateGitAppConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_private_key" "test" {
  name        = "%[1]s-key"
  private_key = <<-EOT
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

resource "coolify_private_git_application" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  private_key_uuid = coolify_private_key.test.uuid
  git_repository   = "git@github.com:coollabsio/coolify-examples.git"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  %[3]s
}
`, name, serverUUID, extra)
}
