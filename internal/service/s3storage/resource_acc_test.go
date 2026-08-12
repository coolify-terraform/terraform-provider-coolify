package s3storage_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccS3StorageResource_CRUD exercises coolify_s3_storage against a real
// Coolify instance with S3StoragesController (Coolify >= v4.3.0).
func TestAccS3StorageResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfNoS3StorageAPI(t)

	name := acctest.RandomWithPrefix("tf-acc-s3")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_s3_storage", "/api/v1/s3-storages/"),
		Steps: []resource.TestStep{
			{
				Config: testAccS3StorageConfig(name, "us-east-1", "acc-bucket"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_s3_storage.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", name),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "acc-bucket"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "us-east-1"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "http://coolify-minio:9000"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "key", "minioadmin"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "secret", "minioadmin123"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "is_usable", "false"),
				),
			},
			{
				Config:             testAccS3StorageConfig(name, "us-east-1", "acc-bucket"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccS3StorageConfig(name+"-upd", "eu-west-1", "acc-bucket-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", name+"-upd"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "acc-bucket-2"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "eu-west-1"),
				),
			},
			{
				ResourceName:                         "coolify_s3_storage.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_s3_storage.test", "uuid"),
				// Coolify hides key/secret unless the token can read sensitive fields.
				ImportStateVerifyIgnore: []string{"key", "secret"},
			},
		},
	})
}

// TestAccS3StorageDataSources exercises singular and list data sources.
func TestAccS3StorageDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfNoS3StorageAPI(t)

	name := acctest.RandomWithPrefix("tf-acc-s3-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_s3_storage", "/api/v1/s3-storages/"),
		Steps: []resource.TestStep{
			{
				Config: testAccS3StorageDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coolify_s3_storage.by_uuid", "uuid",
						"coolify_s3_storage.test", "uuid",
					),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.by_uuid", "name", name),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.by_uuid", "bucket", "ds-bucket"),
					resource.TestCheckResourceAttrSet("data.coolify_s3_storages.all", "s3_storages.#"),
				),
			},
		},
	})
}

func testAccS3StorageConfig(name, region, bucket string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_s3_storage" "test" {
  name        = %[1]q
  description = "acceptance test"
  endpoint    = "http://coolify-minio:9000"
  bucket      = %[3]q
  region      = %[2]q
  key         = "minioadmin"
  secret      = "minioadmin123"
}
`, name, region, bucket)
}

func testAccS3StorageDataSourceConfig(name string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_s3_storage" "test" {
  name     = %[1]q
  endpoint = "http://coolify-minio:9000"
  bucket   = "ds-bucket"
  region   = "us-east-1"
  key      = "minioadmin"
  secret   = "minioadmin123"
}

data "coolify_s3_storage" "by_uuid" {
  uuid = coolify_s3_storage.test.uuid
}

data "coolify_s3_storages" "all" {
  depends_on = [coolify_s3_storage.test]
  filter {
    name   = "name"
    values = [%[1]q]
  }
}
`, name)
}
