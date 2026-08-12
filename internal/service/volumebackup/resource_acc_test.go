package volumebackup_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccStorageBackupResource_CRUD exercises coolify_storage_backup against a
// real Coolify instance with VolumeBackupsController (Coolify >= v4.3.0;
// coollabsio/coolify#10946). CI boots coollabsio/coolify:edge.
func TestAccStorageBackupResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfNoVolumeBackupAPI(t)

	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-volbkp")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             testAccStorageBackupDestroy,
		Steps: []resource.TestStep{
			// Create schedule on a disposable application storage
			{
				Config: testAccStorageBackupConfig(name, serverUUID, "0 2 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_storage_backup.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_storage_backup.test", "storage_uuid"),
					resource.TestCheckResourceAttrSet("coolify_storage_backup.test", "application_uuid"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "storage_type", "persistent"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "retention_amount_locally", "7"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "timeout", "3600"),
				),
			},
			// Idempotency
			{
				Config:             testAccStorageBackupConfig(name, serverUUID, "0 2 * * *"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update frequency
			{
				Config: testAccStorageBackupConfig(name, serverUUID, "0 3 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 3 * * *"),
					resource.TestCheckResourceAttrSet("coolify_storage_backup.test", "uuid"),
				),
			},
			// Import (schedule fields are state-only; no GET on Coolify)
			{
				ResourceName:                         "coolify_storage_backup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "storage_uuid",
				ImportStateIdFunc: testAccStorageBackupImportStateIdFunc(
					"coolify_application_dockerfile.test",
					"coolify_storage.test",
				),
				ImportStateVerifyIgnore: []string{
					"uuid", "frequency", "enabled", "save_s3", "disable_local_backup",
					"stop_during_backup", "s3_storage_uuid", "storage_type",
					"retention_amount_locally", "retention_days_locally", "retention_max_storage_locally",
					"retention_amount_s3", "retention_days_s3", "retention_max_storage_s3", "timeout",
				},
			},
		},
	})
}

// TestAccStorageBackupResource_S3 exercises save_s3 + s3_storage_uuid against
// real Coolify MinIO (COOLIFY_S3_STORAGE_UUID). Requires VolumeBackupsController.
func TestAccStorageBackupResource_S3(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfNoVolumeBackupAPI(t)

	serverUUID := acctest.AccTestServerUUID(t)
	s3UUID := acctest.AccTestS3StorageUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-volbkp-s3")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             testAccStorageBackupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBackupS3Config(name, serverUUID, s3UUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_storage_backup.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "save_s3", "true"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "s3_storage_uuid", s3UUID),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 3 * * *"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "storage_type", "persistent"),
				),
			},
			{
				Config:             testAccStorageBackupS3Config(name, serverUUID, s3UUID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:                         "coolify_storage_backup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "storage_uuid",
				ImportStateIdFunc: testAccStorageBackupImportStateIdFunc(
					"coolify_application_dockerfile.test",
					"coolify_storage.test",
				),
				ImportStateVerifyIgnore: []string{
					"uuid", "frequency", "enabled", "save_s3", "disable_local_backup",
					"stop_during_backup", "s3_storage_uuid", "storage_type",
					"retention_amount_locally", "retention_days_locally", "retention_max_storage_locally",
					"retention_amount_s3", "retention_days_s3", "retention_max_storage_s3", "timeout",
				},
			},
		},
	})
}

func testAccStorageBackupConfig(name, serverUUID, frequency string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_application_dockerfile" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}

resource "coolify_storage" "test" {
  application_uuid = coolify_application_dockerfile.test.uuid
  name             = %[1]q
  mount_path       = "/data"
}

resource "coolify_storage_backup" "test" {
  application_uuid = coolify_application_dockerfile.test.uuid
  storage_uuid     = coolify_storage.test.uuid
  frequency        = %[3]q
  enabled          = true
}
`, name, serverUUID, frequency)
}

func testAccStorageBackupS3Config(name, serverUUID, s3UUID string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_application_dockerfile" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}

resource "coolify_storage" "test" {
  application_uuid = coolify_application_dockerfile.test.uuid
  name             = %[1]q
  mount_path       = "/data"
}

resource "coolify_storage_backup" "test" {
  application_uuid = coolify_application_dockerfile.test.uuid
  storage_uuid     = coolify_storage.test.uuid
  frequency        = "0 3 * * *"
  enabled          = true
  save_s3          = true
  s3_storage_uuid  = %[3]q
}
`, name, serverUUID, s3UUID)
}

func testAccStorageBackupImportStateIdFunc(appResourceName, storageResourceName string) resource.ImportStateIdFunc {
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

// testAccStorageBackupDestroy confirms the schedule is gone. Coolify has no GET
// for volume backup schedules; DELETE returns 404 when missing (or when the
// parent resource/storage is already removed during full destroy).
func testAccStorageBackupDestroy(s *terraform.State) error {
	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	token := os.Getenv("COOLIFY_TOKEN")
	if endpoint == "" || token == "" {
		return fmt.Errorf("COOLIFY_ENDPOINT and COOLIFY_TOKEN required for CheckDestroy")
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "coolify_storage_backup" {
			continue
		}
		appUUID := rs.Primary.Attributes["application_uuid"]
		svcUUID := rs.Primary.Attributes["service_uuid"]
		dbUUID := rs.Primary.Attributes["database_uuid"]
		storUUID := rs.Primary.Attributes["storage_uuid"]
		if storUUID == "" {
			continue
		}

		var path string
		switch {
		case appUUID != "":
			path = fmt.Sprintf("%s/api/v1/applications/%s/storages/%s/backups", endpoint, appUUID, storUUID)
		case svcUUID != "":
			path = fmt.Sprintf("%s/api/v1/services/%s/storages/%s/backups", endpoint, svcUUID, storUUID)
		case dbUUID != "":
			path = fmt.Sprintf("%s/api/v1/databases/%s/storages/%s/backups", endpoint, dbUUID, storUUID)
		default:
			return fmt.Errorf("coolify_storage_backup %s has no parent uuid", rs.Primary.ID)
		}

		req, err := http.NewRequest(http.MethodDelete, path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("DELETE volume backup for %s: %w", storUUID, err)
		}
		_ = resp.Body.Close()

		// 404 = already gone (desired). 200 = delete succeeded late. 409 = in progress.
		switch resp.StatusCode {
		case http.StatusNotFound, http.StatusOK:
			// destroyed
		default:
			return fmt.Errorf("volume backup schedule for storage %s still present (HTTP %d on DELETE)",
				storUUID, resp.StatusCode)
		}
	}
	return nil
}
