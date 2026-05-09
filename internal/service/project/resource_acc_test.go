package project_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectResource_basic(t *testing.T) {
	t.Parallel()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping acceptance test")
	}
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-proj")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name        = %q
  description = "Acceptance test project"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_project.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_project.test", "name", name),
					resource.TestCheckResourceAttr("coolify_project.test", "description", "Acceptance test project"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name        = %q
  description = "Updated description"
}
`, name+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_project.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_project.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:                         "coolify_project.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}
