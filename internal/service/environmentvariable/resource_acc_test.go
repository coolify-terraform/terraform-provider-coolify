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
// TestAccEnvironmentVariableResource_DatabaseCRUD
// ---------------------------------------------------------------------------

func TestAccEnvironmentVariableResource_DatabaseCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-envvar-db")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_environment_variable", "database_uuid", "/api/v1/databases/%s/envs"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccEnvVarDbConfig(name, serverUUID, "db-initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment_variable.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "DB_TEST_VAR"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "db-initial"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "false"),
				),
			},
			// Idempotency check
			{
				Config:             testAccEnvVarDbConfig(name, serverUUID, "db-initial"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update
			{
				Config: testAccEnvVarDbConfig(name, serverUUID, "db-updated"),
				Check:  resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "db-updated"),
			},
			// Import
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"value"},
				ImportStateIdFunc:                    testAccEnvVarDbImportStateIdFunc("coolify_database_postgresql.test", "coolify_environment_variable.test"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccEnvironmentVariableResource_ServiceCRUD
// ---------------------------------------------------------------------------

func TestAccEnvironmentVariableResource_ServiceCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-envvar-svc")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_environment_variable", "service_uuid", "/api/v1/services/%s/envs"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccEnvVarServiceConfig(name, serverUUID, "svc-initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment_variable.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "SVC_TEST_VAR"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "svc-initial"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "false"),
				),
			},
			// Idempotency check
			{
				Config:             testAccEnvVarServiceConfig(name, serverUUID, "svc-initial"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update
			{
				Config: testAccEnvVarServiceConfig(name, serverUUID, "svc-updated"),
				Check:  resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "svc-updated"),
			},
			// Import
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"value"},
				ImportStateIdFunc:                    testAccEnvVarServiceImportStateIdFunc("coolify_service.test", "coolify_environment_variable.test"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccEnvironmentVariableSingularDataSource
// ---------------------------------------------------------------------------

func TestAccEnvironmentVariableSingularDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-envvar-sds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_environment_variable", "application_uuid", "/api/v1/applications/%s/envs"),
		Steps: []resource.TestStep{
			{
				Config: testAccEnvVarAppConfig(name, serverUUID, "singular-ds-value") + `
data "coolify_environment_variable" "test" {
  uuid             = coolify_environment_variable.test.uuid
  application_uuid = coolify_dockerfile_application.test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.coolify_environment_variable.test", "uuid", "coolify_environment_variable.test", "uuid"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "key", "TEST_VAR"),
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

func testAccEnvVarDbConfig(name, serverUUID, value string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_database_postgresql" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
}

resource "coolify_environment_variable" "test" {
  database_uuid = coolify_database_postgresql.test.uuid
  key           = "DB_TEST_VAR"
  value         = %[3]q
}
`, name, serverUUID, value)
}

func testAccEnvVarServiceConfig(name, serverUUID, value string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  type         = "uptime-kuma"
}

resource "coolify_environment_variable" "test" {
  service_uuid = coolify_service.test.uuid
  key          = "SVC_TEST_VAR"
  value        = %[3]q
}
`, name, serverUUID, value)
}

func testAccEnvVarDbImportStateIdFunc(dbResourceName, envVarResourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		dbRS, ok := s.RootModule().Resources[dbResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", dbResourceName)
		}
		envRS, ok := s.RootModule().Resources[envVarResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", envVarResourceName)
		}
		return fmt.Sprintf("database:%s:%s", dbRS.Primary.Attributes["uuid"], envRS.Primary.Attributes["uuid"]), nil
	}
}

func testAccEnvVarServiceImportStateIdFunc(svcResourceName, envVarResourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		svcRS, ok := s.RootModule().Resources[svcResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", svcResourceName)
		}
		envRS, ok := s.RootModule().Resources[envVarResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", envVarResourceName)
		}
		return fmt.Sprintf("service:%s:%s", svcRS.Primary.Attributes["uuid"], envRS.Primary.Attributes["uuid"]), nil
	}
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
