package notificationemail_test

import (
	"context"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
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

// TestAccEmailNotificationResource_SMTPEhloDomain writes smtp_ehlo_domain
// (Coolify tip after #11398). Soft-skip when the instance version string is
// below 4.3.10. Do not use AccTestSkipIfCoolifyBelow: CI edge often reports
// 4.3.0 and COOLIFY_REQUIRE_TIP_APIS=1 would fail a 4.3.10 floor.
func TestAccEmailNotificationResource_SMTPEhloDomain(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)
	c := acctest.AccTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ver, err := c.GetVersion(ctx)
	if err != nil {
		t.Fatalf("reading Coolify version: %v", err)
	}
	if !client.IsVersionAtLeast(ver, "4.3.10") {
		t.Skipf("smtp_ehlo_domain needs Coolify >= 4.3.10 (instance reports %s)", ver)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_email" "test" {
  smtp_enabled               = false
  resend_enabled             = false
  use_instance_email_settings = false
  smtp_ehlo_domain           = "mail.example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_ehlo_domain", "mail.example.com"),
				),
			},
		},
	})
}
