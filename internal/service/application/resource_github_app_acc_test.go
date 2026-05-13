package application_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGitHubAppApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	if os.Getenv("COOLIFY_GITHUB_APP_UUID") == "" {
		t.Skip("COOLIFY_GITHUB_APP_UUID not set, skipping (real GitHub App credentials required)")
	}
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp-app")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_github_app_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccGitHubAppApplicationConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_github_app_application.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_github_app_application.test", "ports_exposes", "3000"),
				),
			},
			// Step 2: Update description
			{
				Config: testAccGitHubAppApplicationConfig(name, serverUUID, `description = "Updated github app application"`),
				Check:  resource.TestCheckResourceAttr("coolify_github_app_application.test", "description", "Updated github app application"),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_github_app_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_github_app_application.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name", "github_app_uuid"},
			},
		},
	})
}

func testAccGitHubAppApplicationConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_github_app" "test" {
  name             = "%[1]s-ghapp"
  app_id           = 12345
  installation_id  = 67890
  client_id        = "Iv1.fake123456789"
  client_secret    = "fake-client-secret-value"
  private_key_uuid = "pk-uuid-acctest"
}

resource "coolify_github_app_application" "test" {
  project_uuid    = coolify_project.test.uuid
  server_uuid     = %[2]q
  github_app_uuid = coolify_github_app.test.uuid
  git_repository  = "coollabsio/coolify-examples"
  build_pack      = "nixpacks"
  ports_exposes   = "3000"
  %[3]s
}
`, name, serverUUID, extra)
}
