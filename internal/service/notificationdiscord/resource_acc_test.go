package notificationdiscord_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDiscordNotificationResource exercises GET/PATCH /notifications/discord
// (Coolify >= v4.3.0). Uses a public example.com webhook so SafeWebhookUrl passes;
// enabled stays false so Coolify does not attempt delivery during the test.
func TestAccDiscordNotificationResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_discord" "test" {
  enabled            = false
  webhook_url        = "https://example.com/coolify-tf-acc-discord"
  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "webhook_url", "https://example.com/coolify-tf-acc-discord"),
					resource.TestCheckResourceAttr("coolify_notification_discord.test", "deployment_failure", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_discord" "test" {
  enabled            = false
  webhook_url        = "https://example.com/coolify-tf-acc-discord"
  deployment_failure = true
  backup_failure     = false
  server_disk_usage  = true
}
`,
				Check: resource.TestCheckResourceAttr("coolify_notification_discord.test", "backup_failure", "false"),
			},
			{
				ResourceName:            "coolify_notification_discord.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
		},
	})
}
