package application_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIntegration_FullStack creates a full resource dependency chain and
// verifies Terraform's dependency ordering works against a real Coolify
// instance, including destroy ordering.
//
//	project -> dockerfile_application -> env_var
//	                                  -> storage
//	                                  -> scheduled_task
//	        -> postgresql_database    -> database_backup
func TestAccIntegration_FullStack(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-full")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: testAccFullStackConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Project
					resource.TestCheckResourceAttrSet("coolify_project.stack", "uuid"),
					// Application
					resource.TestCheckResourceAttrSet("coolify_dockerfile_application.stack", "uuid"),
					// Env var
					resource.TestCheckResourceAttrSet("coolify_environment_variable.stack", "uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.stack", "key", "STACK_VAR"),
					// Storage
					resource.TestCheckResourceAttrSet("coolify_storage.stack", "uuid"),
					resource.TestCheckResourceAttr("coolify_storage.stack", "mount_path", "/data"),
					// Scheduled task
					resource.TestCheckResourceAttrSet("coolify_scheduled_task.stack", "uuid"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.stack", "command", "echo stack"),
					// Database
					resource.TestCheckResourceAttrSet("coolify_database_postgresql.stack", "uuid"),
					// Backup
					resource.TestCheckResourceAttrSet("coolify_database_backup.stack", "uuid"),
					resource.TestCheckResourceAttr("coolify_database_backup.stack", "frequency", "0 3 * * *"),
				),
			},
			// Idempotency check on the full stack
			{
				Config:             testAccFullStackConfig(name, serverUUID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccFullStackConfig(name, serverUUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "stack" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "stack" {
  project_uuid        = coolify_project.stack.uuid
  server_uuid         = %[2]q
  name                = "%[1]s-app"
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}

resource "coolify_environment_variable" "stack" {
  application_uuid = coolify_dockerfile_application.stack.uuid
  key              = "STACK_VAR"
  value            = "stack-value"
}

resource "coolify_storage" "stack" {
  application_uuid = coolify_dockerfile_application.stack.uuid
  name             = "%[1]s-vol"
  mount_path       = "/data"
}

resource "coolify_scheduled_task" "stack" {
  application_uuid = coolify_dockerfile_application.stack.uuid
  name             = "%[1]s-task"
  command          = "echo stack"
  frequency        = "0 * * * *"
}

resource "coolify_database_postgresql" "stack" {
  project_uuid = coolify_project.stack.uuid
  server_uuid  = %[2]q
  name         = "%[1]s-db"
}

resource "coolify_database_backup" "stack" {
  database_uuid         = coolify_database_postgresql.stack.uuid
  frequency             = "0 3 * * *"
  enabled               = true
  retain_amount_locally = 3
}
`, name, serverUUID)
}
