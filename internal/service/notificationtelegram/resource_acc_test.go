package notificationtelegram_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTelegramNotificationResource exercises GET/PATCH /notifications/telegram
// (Coolify >= v4.3.0 only).
func TestAccTelegramNotificationResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_telegram" "test" {
  enabled            = false
  token              = "0000000000:TFACCPLACEHOLDERTOKEN"
  chat_id            = "-1000000000000"
  deployment_failure = true
  backup_failure     = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "deployment_failure", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_telegram" "test" {
  enabled            = false
  token              = "0000000000:TFACCPLACEHOLDERTOKEN"
  chat_id            = "-1000000000000"
  deployment_failure = true
  backup_failure     = false
  status_change      = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "backup_failure", "false"),
					resource.TestCheckResourceAttr("coolify_notification_telegram.test", "status_change", "true"),
				),
			},
			{
				ResourceName:            "coolify_notification_telegram.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "chat_id"},
			},
		},
	})
}
