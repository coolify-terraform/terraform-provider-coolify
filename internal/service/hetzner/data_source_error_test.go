package hetzner_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestHetznerListDataSources_APIError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		hclType string
		errRe   string
	}{
		{"images", "coolify_hetzner_images", `Error listing Hetzner images`},
		{"locations", "coolify_hetzner_locations", `Error listing Hetzner locations`},
		{"server_types", "coolify_hetzner_server_types", `Error listing Hetzner server types`},
		{"ssh_keys", "coolify_hetzner_ssh_keys", `Error listing Hetzner SSH keys`},
		{"firewalls", "coolify_hetzner_firewalls", `Error listing Hetzner firewalls`},
		{"networks", "coolify_hetzner_networks", `Error listing Hetzner networks`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
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
