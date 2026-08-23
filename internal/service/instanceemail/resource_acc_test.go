package instanceemail_test

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func accSkipIfNoInstanceEmailAPI(t *testing.T) {
	t.Helper()
	acctest.TestAccPreCheck(t)

	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	token := os.Getenv("COOLIFY_TOKEN")
	path := endpoint + "/api/v1/settings/email"
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("building instance email API probe request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("instance email API probe failed (cannot reach Coolify): %v", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		return
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		t.Skipf("Coolify instance has no InstanceEmailSettingsController (HTTP %d). Need Coolify >= v4.3.10. Body: %s",
			resp.StatusCode, string(body))
	default:
		if resp.StatusCode >= 500 {
			t.Skipf("instance email API probe returned HTTP %d: %s", resp.StatusCode, string(body))
		}
	}
}

// TestAccInstanceEmailSettingsResource exercises GET/PATCH /settings/email.
// Soft-skip when the route is missing. Do not use AccTestSkipIfCoolifyBelow(4.3.10):
// CI edge often reports 4.3.0. smtp_ehlo_domain is extra-key probed, not
// version-gated.
func TestAccInstanceEmailSettingsResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	accSkipIfNoInstanceEmailAPI(t)

	steps := []resource.TestStep{
		{
			Config: acctest.ConfigProviderBlock() + `
resource "coolify_instance_email_settings" "test" {
  smtp_enabled    = false
  resend_enabled  = false
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "id", "current"),
				resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_enabled", "false"),
				resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "resend_enabled", "false"),
			),
		},
	}
	if acctest.AccTestSMTPEhloDomainAccepted(t) {
		steps = append(steps, resource.TestStep{
			Config: acctest.ConfigProviderBlock() + `
resource "coolify_instance_email_settings" "test" {
  smtp_enabled     = false
  resend_enabled   = false
  smtp_ehlo_domain = "mail.example.com"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_ehlo_domain", "mail.example.com"),
			),
		})
	}
	steps = append(steps, resource.TestStep{
		ResourceName:      "coolify_instance_email_settings.test",
		ImportState:       true,
		ImportStateId:     "current",
		ImportStateVerify: true,
		ImportStateVerifyIgnore: []string{
			"smtp_from_address", "smtp_from_name", "smtp_host",
			"smtp_username", "smtp_password", "resend_api_key",
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps:                    steps,
	})
}
