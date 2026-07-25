package digitalocean_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDigitalOceanRegionsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoDigitalOceanToken(t)
	name := acctest.RandomWithPrefix("tf-acc-do-reg")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccDigitalOceanCloudTokenConfig(name) + `
data "coolify_digitalocean_regions" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.do_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_digitalocean_regions.test", "regions.#"),
		}},
	})
}

func TestAccDigitalOceanSizesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoDigitalOceanToken(t)
	name := acctest.RandomWithPrefix("tf-acc-do-sz")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccDigitalOceanCloudTokenConfig(name) + `
data "coolify_digitalocean_sizes" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.do_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_digitalocean_sizes.test", "sizes.#"),
		}},
	})
}

func TestAccDigitalOceanImagesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoDigitalOceanToken(t)
	name := acctest.RandomWithPrefix("tf-acc-do-img")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccDigitalOceanCloudTokenConfig(name) + `
data "coolify_digitalocean_images" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.do_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_digitalocean_images.test", "images.#"),
		}},
	})
}

func TestAccDigitalOceanSSHKeysDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoDigitalOceanToken(t)
	name := acctest.RandomWithPrefix("tf-acc-do-ssh")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + testAccDigitalOceanCloudTokenConfig(name) + `
data "coolify_digitalocean_ssh_keys" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.do_test.uuid
}
`,
			Check: resource.TestCheckResourceAttrSet("data.coolify_digitalocean_ssh_keys.test", "ssh_keys.#"),
		}},
	})
}
