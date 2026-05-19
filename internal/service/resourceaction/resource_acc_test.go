package resourceaction_test

import (
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceActionResource_StartDatabase(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ra-start")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceActionDatabaseConfig(name, serverUUID, "start", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_resource_action.test", "resource_uuid"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "resource_type", "database"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "action", "start"),
				),
			},
		},
	})
}

func TestAccResourceActionResource_StopDatabase(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ra-stop")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceActionDeployedDatabaseConfig(name, serverUUID, "stop", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_resource_action.test", "resource_uuid"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "resource_type", "database"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "action", "stop"),
				),
			},
		},
	})
}

func TestAccResourceActionResource_RestartDatabase(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ra-restart")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceActionDatabaseConfig(name, serverUUID, "restart", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_resource_action.test", "resource_uuid"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "resource_type", "database"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "action", "restart"),
				),
			},
		},
	})
}

func TestAccResourceActionResource_TriggersForceReplace(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ra-trig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_project", "/api/v1/projects/"),
		Steps: []resource.TestStep{
			// Create with initial triggers.
			{
				Config: testAccResourceActionDatabaseConfig(name, serverUUID, "start", map[string]string{"run": "1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_resource_action.test", "action", "start"),
					resource.TestCheckResourceAttr("coolify_resource_action.test", "triggers.run", "1"),
				),
			},
			// Change triggers value to force re-execution (destroy+create).
			{
				Config: testAccResourceActionDatabaseConfig(name, serverUUID, "start", map[string]string{"run": "2"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_resource_action.test", "triggers.run", "2"),
				),
			},
		},
	})
}

// testAccResourceActionDeployedDatabaseConfig creates a database with instant_deploy = true
// so the container is running before the action executes. Used for stop tests.
func testAccResourceActionDeployedDatabaseConfig(name, serverUUID, action string, triggers map[string]string) string {
	triggerBlock := ""
	if len(triggers) > 0 {
		triggerBlock = "  triggers = {\n"
		for k, v := range triggers {
			triggerBlock += fmt.Sprintf("    %s = %q\n", k, v)
		}
		triggerBlock += "  }\n"
	}

	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_database_postgresql" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  environment_name = "production"
  instant_deploy   = true
}

resource "coolify_resource_action" "test" {
  resource_uuid = coolify_database_postgresql.test.uuid
  resource_type = "database"
  action        = %[3]q
%[4]s}
`, name, serverUUID, action, triggerBlock)
}

func testAccResourceActionDatabaseConfig(name, serverUUID, action string, triggers map[string]string) string {
	triggerBlock := ""
	if len(triggers) > 0 {
		triggerBlock = "  triggers = {\n"
		for k, v := range triggers {
			triggerBlock += fmt.Sprintf("    %s = %q\n", k, v)
		}
		triggerBlock += "  }\n"
	}

	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_database_postgresql" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  environment_name = "production"
}

resource "coolify_resource_action" "test" {
  resource_uuid = coolify_database_postgresql.test.uuid
  resource_type = "database"
  action        = %[3]q
%[4]s}
`, name, serverUUID, action, triggerBlock)
}
