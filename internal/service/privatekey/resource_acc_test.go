package privatekey_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccPrivateKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIBogIBAAJBALRiMLH00a6VK6aBIOxSjDQ3cBcVSaDXGfhLzJRNFp+t4/AOeDmR
5jXIx7DKXAIF9LRPz2gMjTb4i/r20hEh4cUCAwEAAQJBAJmHpJzk0fzYIYv3ihEE
3Ni7SIsMFCEzW0MREqYoLfpyBenGChQVqBqy9XAEiTHDhVsMb0ygDVRGGBGk0nkC
IQDjBCIEaeWV//pZGeJBU6o3JRxJV0rYpAf+0JCuXxvhfwIhAMuaVUIzMgEeClUH
7MYhb91EjG7RQcU0fYq+mUKPXaQfAiAXgdSDZvGhRHrFHGMLCcGI0EdCxKNcUmOb
sDijzrCVlQIgbbsHtPPG0oFkkRe8Y+FRZFyJBLlaCRxNyOWLzRYW/BsCIBFJ2Pla
0EQ2/JWFj1fOfsMnVMxOa2A1SL4lXEm6iNgV
-----END RSA PRIVATE KEY-----`

func TestAccPrivateKeyResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-pk")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = %q
  description = "acc test key"
  private_key = <<-EOT
%sEOT
}
`, name, testAccPrivateKeyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", name),
					resource.TestCheckResourceAttr("coolify_private_key.test", "description", "acc test key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "is_git_related", "false"),
				),
			},
			// Step 2: Update name and description
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = %q
  description = "updated acc test key"
  private_key = <<-EOT
%sEOT
}
`, name+"-updated", testAccPrivateKeyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "description", "updated acc test key"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_private_key.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"private_key"},
			},
		},
	})
}
