package environmentvariable_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestAccEnvironmentVariableResource_ApplicationCRUD
// ---------------------------------------------------------------------------

func TestAccEnvironmentVariableResource_ApplicationCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-envvar")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_environment_variable", "application_uuid", "/api/v1/applications/%s/envs"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccEnvVarAppConfig(name, serverUUID, "initial-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment_variable.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "TEST_VAR"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "initial-value"),
				),
			},
			// Step 2: Idempotency - value preserved after read-back
			{
				Config:             testAccEnvVarAppConfig(name, serverUUID, "initial-value"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 3: Update value
			{
				Config: testAccEnvVarAppConfig(name, serverUUID, "updated-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "TEST_VAR"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "updated-value"),
				),
			},
			// Step 4: Idempotency after update - value preserved
			{
				Config:             testAccEnvVarAppConfig(name, serverUUID, "updated-value"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 5: Import
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"value"},
				ImportStateIdFunc:                    testAccEnvVarImportStateIdFunc("coolify_dockerfile_application.test", "coolify_environment_variable.test"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccEnvironmentVariableDataSources
// ---------------------------------------------------------------------------

func TestAccEnvironmentVariableDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-envvar-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_environment_variable", "application_uuid", "/api/v1/applications/%s/envs"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}

resource "coolify_environment_variable" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  key              = "TEST_DS_VAR"
  value            = "ds-value"
}

data "coolify_environment_variables" "by_app" {
  application_uuid = coolify_dockerfile_application.test.uuid
  depends_on       = [coolify_environment_variable.test]
}
`, name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_environment_variables.by_app", "environment_variables.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccEnvVarAppConfig(name, serverUUID, value string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}

resource "coolify_environment_variable" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  key              = "TEST_VAR"
  value            = %[3]q
}
`, name, serverUUID, value)
}

func testAccEnvVarImportStateIdFunc(appResourceName, envVarResourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		appRS, ok := s.RootModule().Resources[appResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", appResourceName)
		}
		envRS, ok := s.RootModule().Resources[envVarResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", envVarResourceName)
		}
		appUUID := appRS.Primary.Attributes["uuid"]
		envUUID := envRS.Primary.Attributes["uuid"]
		return fmt.Sprintf("application:%s:%s", appUUID, envUUID), nil
	}
}
