package cloudtoken_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudTokenListDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

data "coolify_cloud_tokens" "filtered" {
  depends_on = [coolify_cloud_token.first, coolify_cloud_token.second]
  filter {
    name   = "cloud_provider"
    values = ["hetzner"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_cloud_tokens.filtered", "cloud_tokens.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_cloud_tokens.filtered", "cloud_tokens.0.name", "second-token"),
				),
			},
		},
	})
}

func TestCloudTokenListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_cloud_tokens" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing cloud tokens`),
			},
		},
	})
}
