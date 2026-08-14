package applicationdestination_test

import (
	"fmt"
	"testing"

	"regexp"

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
			// Coolify rejects an extra destination on the same server as the
			// application's primary destination. CI has one localhost server.
			ExpectError: regexp.MustCompile(`same server as the primary destination`),
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
		}},
	})
}

func TestAccApplicationDestinationResource_AttachSecondServer(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	serverUUID := acctest.AccTestServerUUID(t)
	secondUUID := acctest.AccTestSecondServerUUID(t)
	proj := acctest.RandomWithPrefix("tf-acc-appdest2")
	net := acctest.RandomWithPrefix("tfnet2")

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
`, proj, secondUUID, net, net, serverUUID),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_application_destination.test", "id"),
				resource.TestCheckResourceAttrSet("coolify_application_destination.test", "destination_uuid"),
			),
		}},
	})
}
