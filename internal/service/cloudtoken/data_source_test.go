package cloudtoken_test

import (
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudTokenDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_cloud_token" "source" {
  name           = "ds-test-token"
  cloud_provider = "aws"
  token          = "ds-secret"
}

data "coolify_cloud_token" "test" {
  uuid = coolify_cloud_token.source.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coolify_cloud_token.test", "uuid",
						"coolify_cloud_token.source", "uuid",
					),
					resource.TestCheckResourceAttr("data.coolify_cloud_token.test", "name", "ds-test-token"),
					resource.TestCheckResourceAttr("data.coolify_cloud_token.test", "cloud_provider", "aws"),
				),
			},
		},
	})
}
