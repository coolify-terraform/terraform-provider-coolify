package application_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateGitApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-privgit")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_private_git", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccPrivateGitAppConfig(name, serverUUID, privKey, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_private_git.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_private_git.test", "git_repository", "git@github.com:coollabsio/coolify-examples.git"),
					resource.TestCheckResourceAttr("coolify_application_private_git.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_application_private_git.test", "ports_exposes", "3000"),
				),
			},
			// Idempotency check
			{
				Config:             testAccPrivateGitAppConfig(name, serverUUID, privKey, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update description
			{
				Config: testAccPrivateGitAppConfig(name, serverUUID, privKey, `description = "Updated private git app"`),
				Check:  resource.TestCheckResourceAttr("coolify_application_private_git.test", "description", "Updated private git app"),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_application_private_git.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_private_git.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name", "private_key_uuid", "project_uuid", "server_uuid"},
			},
		},
	})
}

func testAccPrivateGitAppConfig(name, serverUUID, privKey, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_private_key" "test" {
  name        = "%[1]s-key"
  private_key = %[3]q
}

resource "coolify_application_private_git" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  private_key_uuid = coolify_private_key.test.uuid
  git_repository   = "git@github.com:coollabsio/coolify-examples.git"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  %[4]s
}
`, name, serverUUID, privKey, extra)
}
