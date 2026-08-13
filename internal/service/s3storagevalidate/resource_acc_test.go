package s3storagevalidate_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccS3StorageValidate_Basic exercises coolify_s3_storage_validate against
// a real Coolify instance (Coolify >= v4.3.0).
//
// Creates a disposable public-endpoint S3 config and validates it. Coolify's
// validate call typically fails connectivity for placeholder credentials; the
// resource must surface that as an apply error. Success paths are covered by
// unit tests with httptest mocks.
//
// Intentionally does NOT use COOLIFY_S3_STORAGE_UUID (bootstrap minio-test):
// POST validate on that UUID fails SafeWebhookUrl for host coolify-minio and
// flips is_usable to false, breaking volume/database S3 backup acceptance.
func TestAccS3StorageValidate_Basic(t *testing.T) {
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfNoS3StorageAPI(t)

	name := acctest.RandomWithPrefix("tf-acc-s3-val")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccS3StorageValidateConfig(name),
				// Placeholder AWS credentials almost always fail Coolify's
				// connectivity check; the action resource must still be wired.
				ExpectError: regexp.MustCompile(`(?i)s3 storage validation failed`),
			},
		},
	})
}

func testAccS3StorageValidateConfig(name string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_s3_storage" "test" {
  name     = %q
  endpoint = "https://s3.us-east-1.amazonaws.com"
  bucket   = "tf-acc-validate-bucket"
  region   = "us-east-1"
  key      = "AKIA_TF_ACC_NOT_REAL"
  secret   = "tf-acc-not-a-real-secret"
}

resource "coolify_s3_storage_validate" "test" {
  s3_storage_uuid = coolify_s3_storage.test.uuid
}
`, name)
}
