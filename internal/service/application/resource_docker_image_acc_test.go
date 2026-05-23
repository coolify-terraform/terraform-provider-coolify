package application_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
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
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_docker_image.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "name", name),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:alpine"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "80"),
				),
			},
			// Idempotency check
			{
				Config:             testAccDockerImageAppConfig(name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update description
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_application_docker_image.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_docker_image.test", "uuid"),
				// docker_image is ignored because Coolify may strip the tag (e.g. "nginx:alpine" → "nginx")
				// and the provider cannot reconstruct the original tag on import.
				ImportStateVerifyIgnore: []string{"environment_name", "project_uuid", "server_uuid", "docker_image"},
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

resource "coolify_application_docker_image" "test" {
  project_uuid  = coolify_project.test.uuid
  server_uuid   = %[2]q
  name          = %[1]q
  docker_image  = "nginx:alpine"
  ports_exposes = "80"
  %[3]s
}
`, name, serverUUID, extra)
}
