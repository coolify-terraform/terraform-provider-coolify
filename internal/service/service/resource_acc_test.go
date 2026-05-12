package service_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResource_CreateImport(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-svc")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  type         = "plausible"
}
`, name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_service.test", "name"),
					resource.TestCheckResourceAttr("coolify_service.test", "environment_name", "production"),
				),
			},
			// Import
			{
				ResourceName:                         "coolify_service.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"project_uuid", "server_uuid", "environment_name", "type"},
			},
		},
	})
}

func TestAccServiceDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-svc-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  type         = "plausible"
}

data "coolify_service" "by_uuid" {
  uuid = coolify_service.test.uuid
}

data "coolify_services" "all" {
  depends_on = [coolify_service.test]
}
`, name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify singular data source
					resource.TestCheckResourceAttrPair("data.coolify_service.by_uuid", "uuid", "coolify_service.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_service.by_uuid", "name", "coolify_service.test", "name"),
					// Verify list data source
					resource.TestMatchResourceAttr("data.coolify_services.all", "services.#", regexp.MustCompile(`[1-9]\d*`)),
				),
			},
		},
	})
}
