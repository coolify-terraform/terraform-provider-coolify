package applicationdestination_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationDestinationResource_Attach(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	serverUUID := acctest.AccTestServerUUID(t)
	proj := acctest.RandomWithPrefix("tf-acc-appdest")
	net := acctest.RandomWithPrefix("tfnet")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_destination" "extra" {
  server_uuid = %q
  network     = %q
  name        = %q
  type        = "standalone"
}
resource "coolify_application_dockerfile" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}
resource "coolify_application_destination" "test" {
  application_uuid = coolify_application_dockerfile.test.uuid
  destination_uuid = coolify_destination.extra.uuid
}
`, proj, serverUUID, net, net, serverUUID),
			Check: resource.TestCheckResourceAttrSet("coolify_application_destination.test", "id"),
		}, {
			ResourceName:                         "coolify_application_destination.test",
			ImportState:                          true,
			ImportStateVerify:                    true,
			ImportStateVerifyIdentifierAttribute: "id",
			ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_destination.test", "id"),
		}},
	})
}
