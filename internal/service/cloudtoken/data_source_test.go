package cloudtoken_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
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

func TestCloudTokenDataSource_NotFound(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_cloud_token" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`Error reading cloud token`),
			},
		},
	})
}
