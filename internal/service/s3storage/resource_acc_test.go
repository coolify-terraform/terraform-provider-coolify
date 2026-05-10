package s3storage_test

import (
	"fmt"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccS3StorageResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-s3")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_s3_storage" "test" {
  name       = %q
  endpoint   = "http://minio.local:9000"
  bucket     = "test-bucket"
  region     = "us-east-1"
  access_key = "minioadmin"
  secret_key = "minioadmin"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_s3_storage.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", name),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "http://minio.local:9000"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "test-bucket"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "us-east-1"),
				),
			},
			// Step 2: Update name and description
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_s3_storage" "test" {
  name        = %q
  description = "updated acc test s3"
  endpoint    = "http://minio.local:9000"
  bucket      = "test-bucket"
  region      = "us-east-1"
  access_key  = "minioadmin"
  secret_key  = "minioadmin"
}
`, name+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_s3_storage.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "description", "updated acc test s3"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "http://minio.local:9000"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "test-bucket"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "us-east-1"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_s3_storage.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"access_key", "secret_key"},
			},
		},
	})
}
