package vultr_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestVultrListDataSources_APIError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		hclType string
		errRe   string
	}{
		{"regions", "coolify_vultr_regions", `Error listing Vultr regions`},
		{"plans", "coolify_vultr_plans", `Error listing Vultr plans`},
		{"ssh_keys", "coolify_vultr_ssh_keys", `Error listing Vultr SSH keys`},
		{"os", "coolify_vultr_os", `Error listing Vultr OS`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"message":"Vultr cloud provider token not found."}`, http.StatusNotFound)
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
