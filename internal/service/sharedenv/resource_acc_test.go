package sharedenv_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSharedEnvResource_TeamCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	key := strings.ReplaceAll(acctest.RandomWithPrefix("TF_ACC_SHARED"), "-", "_")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = %q
  value = "on"
}
`, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", key),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = %q
  value = "off"
}
`, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
			{
				ResourceName:                         "coolify_shared_environment_variable.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_shared_environment_variable.test", "id"),
				ImportStateVerifyIgnore:              []string{"value"},
			},
		},
	})
}

func TestAccSharedEnvResource_ProjectCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	key := strings.ReplaceAll(acctest.RandomWithPrefix("TF_ACC_PROJ"), "-", "_")
	proj := acctest.RandomWithPrefix("tf-acc-senv-proj")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_shared_environment_variable" "test" {
  scope        = "project"
  project_uuid = coolify_project.test.uuid
  key          = %q
  value        = "on"
}
`, proj, key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", key),
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "scope", "project"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_shared_environment_variable" "test" {
  scope        = "project"
  project_uuid = coolify_project.test.uuid
  key          = %q
  value        = "off"
}
`, proj, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
		},
	})
}

func TestAccSharedEnvResource_ServerCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	key := strings.ReplaceAll(acctest.RandomWithPrefix("TF_ACC_SRV"), "-", "_")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_shared_environment_variable" "test" {
  scope       = "server"
  server_uuid = %q
  key         = %q
  value       = "on"
}
`, serverUUID, key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", key),
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "scope", "server"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_shared_environment_variable" "test" {
  scope       = "server"
  server_uuid = %q
  key         = %q
  value       = "off"
}
`, serverUUID, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
		},
	})
}

func TestAccSharedEnvResource_EnvironmentCRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	key := strings.ReplaceAll(acctest.RandomWithPrefix("TF_ACC_ENV"), "-", "_")
	proj := acctest.RandomWithPrefix("tf-acc-senv-env")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_shared_environment_variable" "test" {
  scope        = "environment"
  project_uuid = coolify_project.test.uuid
  environment  = "production"
  key          = %q
  value        = "on"
}
`, proj, key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", key),
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "scope", "environment"),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_shared_environment_variable" "test" {
  scope        = "environment"
  project_uuid = coolify_project.test.uuid
  environment  = "production"
  key          = %q
  value        = "off"
}
`, proj, key),
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
		},
	})
}
