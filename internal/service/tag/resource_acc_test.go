package tag_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTagResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	name := acctest.RandomWithPrefix("tf-acc-tag")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_tag" "test" { name = %q }
data "coolify_tag" "by_uuid" { uuid = coolify_tag.test.uuid }
data "coolify_tags" "all" {}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_tag.test", "name", name),
					resource.TestCheckResourceAttrSet("coolify_tag.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_tag.by_uuid", "name", "coolify_tag.test", "name"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_tag" "test" { name = %q }
`, name+"-upd"),
				Check: resource.TestCheckResourceAttr("coolify_tag.test", "name", name+"-upd"),
			},
			{
				ResourceName:                         "coolify_tag.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_tag.test", "uuid"),
			},
		},
	})
}
