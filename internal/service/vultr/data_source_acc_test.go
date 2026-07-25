package vultr_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVultrRegionsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoVultrToken(t)
	name := acctest.RandomWithPrefix("tf-acc-vu-reg")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccVultrCloudTokenConfig(name) + `
data "coolify_vultr_regions" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.vultr_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_vultr_regions.test", "regions.#"),
		}},
	})
}

func TestAccVultrPlansDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoVultrToken(t)
	name := acctest.RandomWithPrefix("tf-acc-vu-pl")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccVultrCloudTokenConfig(name) + `
data "coolify_vultr_plans" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.vultr_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_vultr_plans.test", "plans.#"),
		}},
	})
}

func TestAccVultrOSDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoVultrToken(t)
	name := acctest.RandomWithPrefix("tf-acc-vu-os")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccVultrCloudTokenConfig(name) + `
data "coolify_vultr_os" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.vultr_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_vultr_os.test", "operating_systems.#"),
		}},
	})
}

func TestAccVultrSSHKeysDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoVultrToken(t)
	name := acctest.RandomWithPrefix("tf-acc-vu-ssh")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccVultrCloudTokenConfig(name) + `
data "coolify_vultr_ssh_keys" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.vultr_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_vultr_ssh_keys.test", "ssh_keys.#"),
		}},
	})
}
