package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestAccDockerfileApplicationResource_CRUD
// ---------------------------------------------------------------------------

func TestAccDockerfileApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkrfile")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccDockerfileAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_dockerfile_application.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "dockerfile_location", "/Dockerfile"),
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "ports_exposes", "80"),
				),
			},
			// Step 2: Update description
			{
				Config: testAccDockerfileAppConfig(name, serverUUID, `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_dockerfile_application.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_dockerfile_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccDockerfileApplicationDataSources
// ---------------------------------------------------------------------------

func TestAccDockerfileApplicationDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkrfile-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDockerfileAppWithDataSourcesConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the resource was created
					resource.TestCheckResourceAttrSet("coolify_dockerfile_application.test", "uuid"),
					// Verify the singular data source reads the application
					resource.TestCheckResourceAttrPair(
						"data.coolify_application.test", "uuid",
						"coolify_dockerfile_application.test", "uuid",
					),
					resource.TestCheckResourceAttrPair(
						"data.coolify_application.test", "name",
						"coolify_dockerfile_application.test", "name",
					),
					// Verify the list data source contains at least one application
					resource.TestCheckResourceAttrSet("data.coolify_applications.test", "applications.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccDockerfileAppConfig(name, serverUUID, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  name                = %[1]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
  %[3]s
}
`, name, serverUUID, extra)
}

func testAccDockerfileAppWithDataSourcesConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  name                = %[1]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}

data "coolify_application" "test" {
  uuid = coolify_dockerfile_application.test.uuid
}

data "coolify_applications" "test" {
  depends_on = [coolify_dockerfile_application.test]
}
`, name, serverUUID)
}

// ---------------------------------------------------------------------------
// TestAccApplicationLogsDataSource
// ---------------------------------------------------------------------------

func TestAccApplicationLogsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-applogs")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAppLogsConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_application_logs.test", "logs.#"),
				),
			},
		},
	})
}

func testAccAppLogsConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}
resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  name                = %[1]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}
data "coolify_application_logs" "test" {
  uuid = coolify_dockerfile_application.test.uuid
}
`, name, serverUUID)
}
