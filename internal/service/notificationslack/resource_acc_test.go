package notificationslack_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSlackNotificationResource exercises GET/PATCH /notifications/slack
// (Coolify >= v4.3.0).
func TestAccSlackNotificationResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_slack" "test" {
  enabled            = false
  webhook_url        = "https://example.com/coolify-tf-acc-slack"
  deployment_failure = true
  backup_failure     = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "webhook_url", "https://example.com/coolify-tf-acc-slack"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_slack" "test" {
  enabled            = false
  webhook_url        = "https://example.com/coolify-tf-acc-slack"
  deployment_failure = true
  backup_failure     = false
  status_change      = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "backup_failure", "false"),
					resource.TestCheckResourceAttr("coolify_notification_slack.test", "status_change", "true"),
				),
			},
			{
				ResourceName:            "coolify_notification_slack.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
		},
	})
}
