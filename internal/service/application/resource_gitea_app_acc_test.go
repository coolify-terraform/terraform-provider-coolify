package application_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGiteaAppApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.7.0")
	giteaUUID := strings.TrimSpace(os.Getenv("COOLIFY_GITEA_APP_UUID"))
	repo := strings.TrimSpace(os.Getenv("COOLIFY_GITEA_REPOSITORY"))
	if giteaUUID == "" || repo == "" {
		t.Skip("Gitea application acceptance requires COOLIFY_GITEA_APP_UUID and COOLIFY_GITEA_REPOSITORY")
	}
	branch := strings.TrimSpace(os.Getenv("COOLIFY_GITEA_BRANCH"))
	if branch == "" {
		branch = "main"
	}
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-gitea-app")
	create := testAccGiteaAppConfig(name, serverUUID, giteaUUID, repo, branch, "3000", "")
	updated := testAccGiteaAppConfig(name, serverUUID, giteaUUID, repo, branch, "8080", `description = "updated gitea app"`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_gitea_app", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: create,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_gitea_app.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "gitea_app_uuid", giteaUUID),
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "ports_exposes", "3000"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "description", "updated gitea app"),
					resource.TestCheckResourceAttr("coolify_application_gitea_app.test", "ports_exposes", "8080"),
				),
			},
			{
				ResourceName:                         "coolify_application_gitea_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_gitea_app.test", "uuid"),
				ImportStateVerifyIgnore: []string{
					"environment_name", "gitea_app_uuid", "project_uuid", "server_uuid",
					"dockerfile", "container_present",
				},
			},
		},
	})
}

func testAccGiteaAppConfig(name, serverUUID, giteaUUID, repo, branch, ports, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_gitea_app" "test" {
  project_uuid   = coolify_project.test.uuid
  server_uuid    = %q
  gitea_app_uuid = %q
  git_repository = %q
  git_branch     = %q
  build_pack     = "nixpacks"
  ports_exposes  = %q
  %s
}
`, name, serverUUID, giteaUUID, repo, branch, ports, extra)
}
