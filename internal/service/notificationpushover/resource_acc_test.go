package notificationpushover_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPushoverNotificationResource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.AccTestSkipIfNoNotificationAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_pushover" "test" {
  enabled            = false
  user_key           = "u-coolify-tf-acc"
  api_token          = "a-coolify-tf-acc"
  deployment_failure = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "deployment_failure", "true"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + `
resource "coolify_notification_pushover" "test" {
  enabled            = false
  user_key           = "u-coolify-tf-acc"
  api_token          = "a-coolify-tf-acc"
  deployment_failure = true
  backup_failure     = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_pushover.test", "backup_failure", "true"),
				),
			},
			{
				ResourceName:            "coolify_notification_pushover.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"user_key", "api_token"},
			},
		},
	})
}
