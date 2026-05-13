package scheduledtask_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestAccScheduledTaskResource_CRUD
// ---------------------------------------------------------------------------

func TestAccScheduledTaskResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-schtask")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_scheduled_task", "application_uuid", "/api/v1/applications/%s/scheduled-tasks"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccScheduledTaskConfig(name, serverUUID, "echo hello"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_scheduled_task.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "name", name),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "command", "echo hello"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "frequency", "* * * * *"),
				),
			},
			// Step 2: Update command
			{
				Config: testAccScheduledTaskConfig(name, serverUUID, "echo updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "command", "echo updated"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_scheduled_task.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    testAccScheduledTaskImportStateIdFunc("coolify_dockerfile_application.test", "coolify_scheduled_task.test"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccScheduledTaskDataSources
// ---------------------------------------------------------------------------

func TestAccScheduledTaskDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-schtask-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_scheduled_task", "application_uuid", "/api/v1/applications/%s/scheduled-tasks"),
		Steps: []resource.TestStep{
			{
				Config: testAccScheduledTaskWithDataSourcesConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the resource was created
					resource.TestCheckResourceAttrSet("coolify_scheduled_task.test", "uuid"),
					// Verify the list data source contains at least one task
					resource.TestCheckResourceAttrSet("data.coolify_scheduled_tasks.test", "tasks.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccScheduledTaskConfig(name, serverUUID, command string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}

resource "coolify_scheduled_task" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  name             = %[1]q
  command          = %[3]q
  frequency        = "* * * * *"
}
`, name, serverUUID, command)
}

func testAccScheduledTaskWithDataSourcesConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}

resource "coolify_scheduled_task" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  name             = %[1]q
  command          = "echo hello"
  frequency        = "* * * * *"
}

data "coolify_scheduled_tasks" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  depends_on       = [coolify_scheduled_task.test]
}
`, name, serverUUID)
}

// ---------------------------------------------------------------------------
// TestAccTaskExecutionsDataSource
// ---------------------------------------------------------------------------

func TestAccTaskExecutionsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-taskexec")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckNestedDestroy("coolify_scheduled_task", "application_uuid", "/api/v1/applications/%s/scheduled-tasks"),
		Steps: []resource.TestStep{
			{
				Config: testAccTaskExecutionsConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.coolify_task_executions.test", "executions.#"),
				),
			},
		},
	})
}

func testAccTaskExecutionsConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}
resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}
resource "coolify_scheduled_task" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  name             = %[1]q
  command          = "echo hello"
  frequency        = "0 * * * *"
}
data "coolify_task_executions" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  task_uuid        = coolify_scheduled_task.test.uuid
}
`, name, serverUUID)
}

func testAccScheduledTaskImportStateIdFunc(appResourceName, taskResourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		appRS, ok := s.RootModule().Resources[appResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", appResourceName)
		}
		taskRS, ok := s.RootModule().Resources[taskResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", taskResourceName)
		}
		appUUID := appRS.Primary.Attributes["uuid"]
		taskUUID := taskRS.Primary.Attributes["uuid"]
		return fmt.Sprintf("application:%s:%s", appUUID, taskUUID), nil
	}
}
