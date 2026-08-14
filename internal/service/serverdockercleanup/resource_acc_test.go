package serverdockercleanup_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerDockerCleanupResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_docker_cleanup" "test" {
  server_uuid              = %q
  docker_cleanup_frequency = "@daily"
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "docker_cleanup_frequency", "@daily"),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_docker_cleanup" "test" {
  server_uuid              = %q
  docker_cleanup_frequency = "daily"
}
`, serverUUID),
				Check: resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "docker_cleanup_frequency", "daily"),
			},
			{
				ResourceName:                         "coolify_server_docker_cleanup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"docker_cleanup_frequency"},
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}
