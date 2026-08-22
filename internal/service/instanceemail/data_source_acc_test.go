package instanceemail_test

import (
	"context"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInstanceEmailSettingsDataSource(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	accSkipIfNoInstanceEmailAPI(t)
	c := acctest.AccTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ver, err := c.GetVersion(ctx)
	if err != nil {
		t.Fatalf("reading Coolify version: %v", err)
	}
	if !client.IsVersionAtLeast(ver, "4.3.10") {
		t.Skipf("Coolify %s is below 4.3.10; instance email settings not on this version string", ver)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
data "coolify_instance_email_settings" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "id", "current"),
					resource.TestCheckResourceAttrSet("data.coolify_instance_email_settings.test", "smtp_enabled"),
				),
			},
		},
	})
}
