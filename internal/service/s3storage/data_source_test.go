package s3storage_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestS3StorageDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_s3_storage" "source" {
  name     = "ds-test-storage"
  endpoint = "https://s3.us-east-1.amazonaws.com"
  bucket   = "ds-bucket"
  region   = "us-east-1"
  key      = "ds-key"
  secret   = "ds-secret"
}

data "coolify_s3_storage" "test" {
  uuid = coolify_s3_storage.source.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coolify_s3_storage.test", "uuid",
						"coolify_s3_storage.source", "uuid",
					),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "name", "ds-test-storage"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "bucket", "ds-bucket"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "region", "us-east-1"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "endpoint", "https://s3.us-east-1.amazonaws.com"),
				),
			},
		},
	})
}

func TestS3StorageDataSource_NotFound(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_s3_storage" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`Error reading S3 storage`),
			},
		},
	})
}
