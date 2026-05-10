package cloudtoken_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudTokenListDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "first" {
  name           = "first-token"
  cloud_provider = "aws"
  token          = "secret-1"
}

resource "coolify_cloud_token" "second" {
  name           = "second-token"
  cloud_provider = "hetzner"
  token          = "secret-2"
}

data "coolify_cloud_tokens" "all" {
  depends_on = [coolify_cloud_token.first, coolify_cloud_token.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_cloud_tokens.all", "cloud_tokens.#", "2"),
				),
			},
		},
	})
}
