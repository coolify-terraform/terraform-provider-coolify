package applicationpreview_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationPreview_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-preview")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationPreviewConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_preview.test", "application_uuid"),
					resource.TestCheckResourceAttr("coolify_application_preview.test", "pull_request_id", "1"),
				),
			},
		},
	})
}

func testAccApplicationPreviewConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %[1]q }
resource "coolify_application_dockerfile" "test" {
  project_uuid    = coolify_project.test.uuid
  server_uuid     = %[2]q
  name            = %[1]q
  dockerfile_location = base64encode("FROM nginx:alpine\nEXPOSE 80\n")
  ports_exposes   = "80"
}
resource "coolify_application_preview" "test" {
  application_uuid = coolify_application_dockerfile.test.uuid
  pull_request_id  = 1
}
`, name, serverUUID)
}
