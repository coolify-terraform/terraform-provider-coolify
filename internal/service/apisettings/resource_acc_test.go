package apisettings_test

import (
	"strings"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPISettings_Enable(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_api_settings" "test" {
  enabled = true
}
`,
				// Enabling the API requires root team (team 0) permissions.
				// The test verifies either success or a 403 permission error.
				// Note: plan may be empty if API is already enabled on the instance.
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
				),
			},
		},
		// Allow the test to pass even if the API token lacks root permissions.
		ErrorCheck: func(err error) error {
			if err != nil && (strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "not allowed")) {
				t.Skip("API token does not have root team permissions for enable/disable API")
			}
			return err
		},
	})
}

func TestAccAPISettings_MCPEnableDisable(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_api_settings" "test" {
  enabled     = true
  mcp_enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_api_settings" "test" {
  enabled     = true
  mcp_enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "false"),
				),
			},
		},
		ErrorCheck: func(err error) error {
			if err != nil && (strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "not allowed")) {
				t.Skip("API token does not have root team permissions for MCP settings")
			}
			return err
		},
	})
}
