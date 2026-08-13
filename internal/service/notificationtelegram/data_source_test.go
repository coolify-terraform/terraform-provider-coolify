package notificationtelegram_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestNotificationDataSource_Read(t *testing.T) {
	t.Parallel()
	store := &mockTelegram{
		ThreadDeployFail: "42",
		EventStore:       notificationcommon.EventStore{DeploymentFailure: true},
	}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_notification_telegram" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_notification_telegram.test", "id", "current"),
					resource.TestCheckResourceAttr("data.coolify_notification_telegram.test", "deployment_failure", "true"),
					resource.TestCheckResourceAttr("data.coolify_notification_telegram.test", "thread_deployment_failure", "42"),
				),
			},
		},
	})
}

func TestNotificationDataSource_ReadAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/telegram", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_notification_telegram" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error reading`),
			},
		},
	})
}
