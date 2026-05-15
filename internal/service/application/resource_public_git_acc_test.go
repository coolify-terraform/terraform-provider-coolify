package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-app")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testAccPublicGitAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_application.test", "ports_exposes", "3000"),
				),
			},
			// Idempotency check
			{
				Config:             testAccPublicGitAppConfig(name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccPublicGitAppConfig(name, serverUUID, `description = "Updated public git app"`),
				Check:  resource.TestCheckResourceAttr("coolify_application.test", "description", "Updated public git app"),
			},
			{
				ResourceName:                         "coolify_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name", "project_uuid", "server_uuid"},
			},
		},
	})
}

func TestAccApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-app-dis")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testAccPublicGitAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application.test", "uuid"),
					acctest.AccCheckResourceDisappears("coolify_application.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccApplicationDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-app-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testAccPublicGitAppConfig(name, serverUUID, "") + `
data "coolify_application" "test" {
  uuid = coolify_application.test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.coolify_application.test", "uuid", "coolify_application.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_application.test", "name", "coolify_application.test", "name"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "build_pack", "nixpacks"),
				),
			},
		},
	})
}

func testAccPublicGitAppConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_application" "test" {
  project_uuid   = coolify_project.test.uuid
  server_uuid    = %[2]q
  name           = %[1]q
  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  %[3]s
}
`, name, serverUUID, extra)
}
