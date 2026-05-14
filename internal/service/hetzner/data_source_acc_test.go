package hetzner_test

import (
	"os"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func skipIfNoHetznerToken(t *testing.T) {
	t.Helper()
	if os.Getenv("COOLIFY_HETZNER_TOKEN") == "" {
		t.Skip("COOLIFY_HETZNER_TOKEN not set, skipping (Coolify proxies requests to Hetzner API)")
	}
}

func testAccHetznerCloudTokenConfig() string {
	return `
resource "coolify_cloud_token" "hetzner_test" {
  name           = "acc-test-hetzner"
  cloud_provider = "hetzner"
  token          = "` + os.Getenv("COOLIFY_HETZNER_TOKEN") + `"
}
`
}

func TestAccHetznerImagesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoHetznerToken(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + testAccHetznerCloudTokenConfig() + `
data "coolify_hetzner_images" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner_test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_hetzner_images.test", "images.#"),
				),
			},
		},
	})
}

func TestAccHetznerLocationsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoHetznerToken(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + testAccHetznerCloudTokenConfig() + `
data "coolify_hetzner_locations" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner_test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_hetzner_locations.test", "locations.#"),
				),
			},
		},
	})
}

func TestAccHetznerServerTypesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoHetznerToken(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + testAccHetznerCloudTokenConfig() + `
data "coolify_hetzner_server_types" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner_test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_hetzner_server_types.test", "server_types.#"),
				),
			},
		},
	})
}

func TestAccHetznerSSHKeysDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoHetznerToken(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + testAccHetznerCloudTokenConfig() + `
data "coolify_hetzner_ssh_keys" "test" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner_test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_hetzner_ssh_keys.test", "ssh_keys.#"),
				),
			},
		},
	})
}
