package server_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-srv")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_server", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccServerConfig(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_server.test", "name", name),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "192.0.2.1"),
					resource.TestCheckResourceAttrSet("coolify_server.test", "private_key_uuid"),
				),
			},
			// Step 2: Update name and description
			{
				Config: testAccServerConfig(name+"-updated", `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_server.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_server.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func testAccServerConfig(name, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
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

resource "coolify_server" "test" {
  name             = %[1]q
  ip               = "192.0.2.1"
  private_key_uuid = coolify_private_key.test.uuid
  %[2]s
}
`, name, extra)
}
