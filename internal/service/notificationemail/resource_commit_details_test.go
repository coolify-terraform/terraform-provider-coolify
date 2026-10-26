package notificationemail_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestEmailNotificationResource_DeploymentCommitDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		send    bool
	}{
		{version: "4.5.0", send: true},
		{version: "4.4.2", send: false},
		{version: "", send: true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			label := tc.version
			if label == "" {
				label = "empty"
			}
			store := &mockEmail{}
			srv := newMockServerVersionMode(store, tc.version, false)
			defer srv.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled = true
  deployment_commit_details_email_notifications = false
`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_notification_email.test", "deployment_commit_details_email_notifications", "false"),
							func(*terraform.State) error {
								store.mu.Lock()
								defer store.mu.Unlock()
								if store.sawCommitDetails != tc.send {
									return fmt.Errorf("Coolify %s saw deployment commit details=%v, want %v", label, store.sawCommitDetails, tc.send)
								}
								if tc.send && (store.DeploymentCommitDetails == nil || *store.DeploymentCommitDetails) {
									return fmt.Errorf("Coolify %s stored %#v, want false", label, store.DeploymentCommitDetails)
								}
								return nil
							},
						),
					},
					{
						Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled = true
  deployment_commit_details_email_notifications = true
`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_notification_email.test", "deployment_commit_details_email_notifications", "true"),
							func(*terraform.State) error {
								store.mu.Lock()
								defer store.mu.Unlock()
								if store.sawCommitDetails != tc.send {
									return fmt.Errorf("Coolify %s update saw deployment commit details=%v, want %v", label, store.sawCommitDetails, tc.send)
								}
								if tc.send && (store.DeploymentCommitDetails == nil || !*store.DeploymentCommitDetails) {
									return fmt.Errorf("Coolify %s update stored %#v, want true", label, store.DeploymentCommitDetails)
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}
