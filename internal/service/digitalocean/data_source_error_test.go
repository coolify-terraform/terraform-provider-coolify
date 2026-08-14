package digitalocean_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDigitalOceanListDataSources_APIError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		hclType string
		errRe   string
	}{
		{"images", "coolify_digitalocean_images", `Error listing DigitalOcean images`},
		{"regions", "coolify_digitalocean_regions", `Error listing DigitalOcean regions`},
		{"sizes", "coolify_digitalocean_sizes", `Error listing DigitalOcean sizes`},
		{"ssh_keys", "coolify_digitalocean_ssh_keys", `Error listing DigitalOcean SSH keys`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"message":"DigitalOcean cloud provider token not found."}`, http.StatusNotFound)
			})))
			t.Cleanup(srv.Close)

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: acctest.ProviderBlockForURL(srv.URL) + `
data "` + tc.hclType + `" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
						ExpectError: regexp.MustCompile(tc.errRe),
					},
				},
			})
		})
	}
}
