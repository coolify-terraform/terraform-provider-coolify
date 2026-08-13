package s3storagevalidate_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccS3StorageValidate_Basic exercises coolify_s3_storage_validate against
// a real Coolify instance (S3StoragesController, Coolify >= v4.3.0). Uses the
// bootstrap minio-test UUID so validation hits a reachable endpoint.
func TestAccS3StorageValidate_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfNoS3StorageAPI(t)

	uuid := acctest.AccTestS3StorageUUID(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccS3StorageValidateExistingConfig(uuid),
				Check: resource.TestCheckResourceAttr(
					"coolify_s3_storage_validate.test", "s3_storage_uuid", uuid),
			},
		},
	})
}

func testAccS3StorageValidateExistingConfig(uuid string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_s3_storage_validate" "test" {
  s3_storage_uuid = %q
}
`, uuid)
}
