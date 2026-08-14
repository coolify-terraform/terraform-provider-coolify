package sharedenv_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSharedEnvResource_TeamCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	key := strings.ReplaceAll(acctest.RandomWithPrefix("TF_ACC_SHARED"), "-", "_")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = %q
  value = "on"
}
`, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", key),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = %q
  value = "off"
}
`, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
			{
				ResourceName:                         "coolify_shared_environment_variable.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_shared_environment_variable.test", "id"),
				ImportStateVerifyIgnore:              []string{"value"},
			},
		},
	})
}
