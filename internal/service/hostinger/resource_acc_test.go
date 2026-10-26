package hostinger_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHostingerServerResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	if os.Getenv("COOLIFY_HOSTINGER_TOKEN") == "" {
		t.Skip("COOLIFY_HOSTINGER_TOKEN not set")
	}
	if os.Getenv("COOLIFY_HOSTINGER_ACC_PURCHASE") == "" {
		t.Skip("COOLIFY_HOSTINGER_ACC_PURCHASE not set; this test buys a VPS and Coolify does not cancel it")
	}
	name := acctest.RandomWithPrefix("tf-acc-hostinger")
	privKey := acctest.GenerateTestRSAKey(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_server_hostinger", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{Config: testAccHostingerConfig(name, privKey, "first"), Check: resource.TestCheckResourceAttrSet("coolify_server_hostinger.test", "uuid")},
			{Config: testAccHostingerConfig(name, privKey, "second"), Check: resource.TestCheckResourceAttr("coolify_server_hostinger.test", "description", "second")},
			{
				ResourceName:                         "coolify_server_hostinger.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_server_hostinger.test", "uuid"),
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid", "item_id", "data_center_id", "template_id",
					"public_key_ids", "post_install_script_id", "enable_backups", "instant_validate", "private_key_uuid",
				},
			},
		},
	})
}

func TestAccHostingerDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	if os.Getenv("COOLIFY_HOSTINGER_TOKEN") == "" {
		t.Skip("COOLIFY_HOSTINGER_TOKEN not set")
	}
	name := acctest.RandomWithPrefix("tf-acc-hostinger-ds")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_token" "hostinger" {
  name           = %q
  cloud_provider = "hostinger"
  token          = %q
}
data "coolify_hostinger_data_centers" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
data "coolify_hostinger_catalog" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
data "coolify_hostinger_templates" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
data "coolify_hostinger_ssh_keys" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
data "coolify_hostinger_post_install_scripts" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
`, name, os.Getenv("COOLIFY_HOSTINGER_TOKEN")),
			Check: resource.TestCheckResourceAttrSet("data.coolify_hostinger_data_centers.test", "data_centers.#"),
		}},
	})
}

func testAccHostingerConfig(name, privKey, description string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_token" "hostinger" {
  name           = "%[1]s-token"
  cloud_provider = "hostinger"
  token          = %[2]q
}
resource "coolify_private_key" "hostinger" {
  name        = "%[1]s-key"
  private_key = %[3]q
}
resource "coolify_server_hostinger" "test" {
  name                       = %[1]q
  description                = %[4]q
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
  private_key_uuid           = coolify_private_key.hostinger.uuid
  item_id                    = "hostingercom-vps-kvm1"
  data_center_id             = 1
  template_id                = 1
}
`, name, os.Getenv("COOLIFY_HOSTINGER_TOKEN"), privKey, description)
}
