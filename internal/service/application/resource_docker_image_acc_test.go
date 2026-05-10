package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestAccDockerImageApplicationResource_CRUD
// ---------------------------------------------------------------------------

func TestAccDockerImageApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkrimg")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_docker_image_application.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "name", name),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "docker_image", "nginx:alpine"),
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "ports_exposes", "80"),
				),
			},
			// Step 2: Update description
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_docker_image_application.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_docker_image_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccDockerImageAppConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_docker_image_application" "test" {
  project_uuid  = coolify_project.test.uuid
  server_uuid   = %[2]q
  name          = %[1]q
  docker_image  = "nginx:alpine"
  ports_exposes = "80"
  %[3]s
}
`, name, serverUUID, extra)
}
