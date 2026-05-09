package provider_test

import (
	"regexp"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProvider_MissingToken(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://localhost:8000"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Missing Coolify Token`),
			},
		},
	})
}

func TestProvider_MissingEndpoint(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  token = "some-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Missing Coolify Endpoint`),
			},
		},
	})
}
