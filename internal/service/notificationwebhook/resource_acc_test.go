package notificationwebhook_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccWebhookNotificationResource exercises GET/PATCH /notifications/webhook
// (Coolify >= v4.3.0).
func TestAccWebhookNotificationResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_webhook" "test" {
  enabled            = false
  webhook_url        = "https://example.com/coolify-tf-acc-webhook"
  deployment_failure = true
  backup_failure     = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "webhook_url", "https://example.com/coolify-tf-acc-webhook"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_webhook" "test" {
  enabled            = false
  webhook_url        = "https://example.com/coolify-tf-acc-webhook"
  deployment_failure = true
  backup_failure     = false
  status_change      = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "backup_failure", "false"),
					resource.TestCheckResourceAttr("coolify_notification_webhook.test", "status_change", "true"),
				),
			},
			{
				ResourceName:            "coolify_notification_webhook.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
		},
	})
}
