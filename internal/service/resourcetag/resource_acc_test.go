package resourcetag_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceTag_Attach(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	name := acctest.RandomWithPrefix("tf-acc-rtag")
	proj := acctest.RandomWithPrefix("tf-acc-rtag-proj")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_application_dockerfile" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}
resource "coolify_tag" "test" { name = %q }
resource "coolify_resource_tag" "test" {
  resource_type = "application"
  resource_uuid = coolify_application_dockerfile.test.uuid
  tag_name      = coolify_tag.test.name
}
`, proj, serverUUID, name),
			Check: resource.TestCheckResourceAttrSet("coolify_resource_tag.test", "tag_uuid"),
		}, {
			ResourceName:                         "coolify_resource_tag.test",
			ImportState:                          true,
			ImportStateVerify:                    true,
			ImportStateVerifyIdentifierAttribute: "id",
			ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_resource_tag.test", "id"),
			ImportStateVerifyIgnore:              []string{"tag_name"},
		}},
	})
}

func TestAccResourceTag_AttachDatabase(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	name := acctest.RandomWithPrefix("tf-acc-rtag-db")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_postgresql", "/api/v1/databases/"),
		Steps: []resource.TestStep{{
			Config: acctest.AccTestDatabaseConfig("coolify_database_postgresql", name, serverUUID, "") + fmt.Sprintf(`
resource "coolify_tag" "test" { name = %q }
resource "coolify_resource_tag" "test" {
  resource_type = "database"
  resource_uuid = coolify_database_postgresql.test.uuid
  tag_name      = coolify_tag.test.name
}
`, name+"-tag"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_resource_tag.test", "tag_uuid"),
				resource.TestCheckResourceAttr("coolify_resource_tag.test", "resource_type", "database"),
			),
		}},
	})
}

func TestAccResourceTag_AttachService(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.2.0")
	name := acctest.RandomWithPrefix("tf-acc-rtag-svc")
	serverUUID := acctest.AccTestServerUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %q }
resource "coolify_service" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %q
  type         = "uptime-kuma"
}
resource "coolify_tag" "test" { name = %q }
resource "coolify_resource_tag" "test" {
  resource_type = "service"
  resource_uuid = coolify_service.test.uuid
  tag_name      = coolify_tag.test.name
}
`, name, serverUUID, name+"-tag"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_resource_tag.test", "tag_uuid"),
				resource.TestCheckResourceAttr("coolify_resource_tag.test", "resource_type", "service"),
			),
		}},
	})
}
