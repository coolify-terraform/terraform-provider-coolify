package cloudinitscript_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudInitScriptResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	name := acctest.RandomWithPrefix("tf-acc-cis")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_init_script" "test" {
  name   = %q
  script = "#cloud-config\npackages: [curl]\n"
}
data "coolify_cloud_init_script" "by_uuid" { uuid = coolify_cloud_init_script.test.uuid }
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_cloud_init_script.test", "name", name),
					resource.TestCheckResourceAttrPair("data.coolify_cloud_init_script.by_uuid", "name", "coolify_cloud_init_script.test", "name"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_init_script" "test" {
  name   = %q
  script = "#cloud-config\npackages: [nginx]\n"
}
`, name+"-upd"),
				Check: resource.TestCheckResourceAttr("coolify_cloud_init_script.test", "name", name+"-upd"),
			},
			{
				ResourceName:                         "coolify_cloud_init_script.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_cloud_init_script.test", "uuid"),
			},
		},
	})
}
