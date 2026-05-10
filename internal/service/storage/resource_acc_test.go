package storage_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestAccStorageResource_CRUD
// ---------------------------------------------------------------------------

func TestAccStorageResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-storage")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccStorageConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_storage.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_storage.test", "name", name),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/data"),
				),
			},
			// Step 2: Update name
			{
				Config: testAccStorageConfig(name+"-updated", serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "name", name+"-updated"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_storage.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    testAccStorageImportStateIdFunc("coolify_dockerfile_application.test", "coolify_storage.test"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccStorageDataSources
// ---------------------------------------------------------------------------

func TestAccStorageDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-storage-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStorageWithDataSourcesConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the resource was created
					resource.TestCheckResourceAttrSet("coolify_storage.test", "uuid"),
					// Verify the list data source contains at least one storage
					resource.TestCheckResourceAttrSet("data.coolify_storages.test", "storages.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccStorageConfig(name, serverUUID, extra string) string {
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

resource "coolify_storage" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  name             = %[1]q
  mount_path       = "/data"
  %[3]s
}
`, name, serverUUID, extra)
}

func testAccStorageWithDataSourcesConfig(name, serverUUID string) string {
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

resource "coolify_storage" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  name             = %[1]q
  mount_path       = "/data"
}

data "coolify_storages" "test" {
  application_uuid = coolify_dockerfile_application.test.uuid
  depends_on       = [coolify_storage.test]
}
`, name, serverUUID)
}

func testAccStorageImportStateIdFunc(appResourceName, storageResourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		appRS, ok := s.RootModule().Resources[appResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", appResourceName)
		}
		storageRS, ok := s.RootModule().Resources[storageResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", storageResourceName)
		}
		appUUID := appRS.Primary.Attributes["uuid"]
		storageUUID := storageRS.Primary.Attributes["uuid"]
		return fmt.Sprintf("application:%s:%s", appUUID, storageUUID), nil
	}
}
