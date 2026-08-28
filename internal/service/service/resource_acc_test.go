package service_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResource_CreateImport(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-svc")
	updated := name + "-upd"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccServiceConfig(name, serverUUID, name, "acc service"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_service.test", "name", name),
					resource.TestCheckResourceAttr("coolify_service.test", "environment_name", "production"),
				),
			},
			// Idempotency check
			{
				Config:             testAccServiceConfig(name, serverUUID, name, "acc service"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update name (PATCH $allowedFields). CreateImport used to skip
			// Update entirely, so extra-key 422 on service PATCH never ran.
			{
				Config: testAccServiceConfig(name, serverUUID, updated, "Updated via acc test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "name", updated),
					resource.TestCheckResourceAttr("coolify_service.test", "description", "Updated via acc test"),
				),
			},
			// Import
			{
				ResourceName:                         "coolify_service.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_service.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"project_uuid", "server_uuid", "environment_name", "type"},
			},
		},
	})
}

func testAccServiceConfig(projectName, serverUUID, serviceName, description string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  type         = "uptime-kuma"
  name         = %[3]q
  description  = %[4]q
}
`, projectName, serverUUID, serviceName, description)
}

func TestAccServiceResource_URLsOrder(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-svc-urls")
	suffix := acctest.RandomWithPrefix("tf")
	zebraURL := fmt.Sprintf("https://zebra-%s.example.com", suffix)
	alphaURL := fmt.Sprintf("https://alpha-%s.example.com", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceURLsOrderConfig(name, serverUUID, zebraURL, alphaURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.#", "2"),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.0.name", "zebra"),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.0.url", zebraURL),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.1.name", "alpha"),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.1.url", alphaURL),
				),
			},
			{
				Config:             testAccServiceURLsOrderConfig(name, serverUUID, zebraURL, alphaURL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccServiceURLsOrderConfig(projectName, serverUUID, zebraURL, alphaURL string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_service" "test" {
  name               = %[1]q
  project_uuid       = coolify_project.test.uuid
  server_uuid        = %[2]q
  instant_deploy     = false
  docker_compose_raw = <<-EOT
services:
  alpha:
    image: nginx:alpine
  zebra:
    image: nginx:alpine
EOT

  urls = [
    { name = "zebra", url = %[3]q },
    { name = "alpha", url = %[4]q },
  ]
}
`, projectName, serverUUID, zebraURL, alphaURL)
}

func TestAccServiceDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-svc-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  type         = "uptime-kuma"
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
					resource.TestCheckResourceAttrPair("data.coolify_service.by_uuid", "status", "coolify_service.test", "status"),
					// Verify list data source
					resource.TestMatchResourceAttr("data.coolify_services.all", "services.#", regexp.MustCompile(`[1-9]\d*`)),
					resource.TestCheckTypeSetElemAttrPair("data.coolify_services.all", "services.*.uuid", "coolify_service.test", "uuid"),
					resource.TestCheckTypeSetElemAttrPair("data.coolify_services.all", "services.*.status", "coolify_service.test", "status"),
				),
			},
		},
	})
}
