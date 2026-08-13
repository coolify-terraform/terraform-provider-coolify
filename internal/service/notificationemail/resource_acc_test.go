package notificationemail_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEmailNotificationResource exercises GET/PATCH /notifications/email
// (Coolify >= v4.3.0 only; skipped on older instances).
func TestAccEmailNotificationResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_email" "test" {
  smtp_enabled               = false
  resend_enabled             = false
  use_instance_email_settings = false
  deployment_failure         = true
  backup_failure             = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "deployment_failure", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_email" "test" {
  smtp_enabled                = false
  resend_enabled              = false
  use_instance_email_settings = false
  deployment_failure          = true
  backup_failure              = false
  server_disk_usage           = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "backup_failure", "false"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "server_disk_usage", "true"),
				),
			},
			{
				ResourceName:      "coolify_notification_email.test",
				ImportState:       true,
				ImportStateId:     "current",
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"smtp_from_address", "smtp_from_name", "smtp_recipients", "smtp_host",
					"smtp_username", "smtp_password", "resend_api_key",
				},
			},
		},
	})
}
