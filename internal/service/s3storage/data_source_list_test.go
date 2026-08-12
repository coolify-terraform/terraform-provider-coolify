package s3storage_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestS3StorageListDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_s3_storage" "first" {
  name     = "first-storage"
  endpoint = "https://s3.us-east-1.amazonaws.com"
  bucket   = "bucket-1"
  region   = "us-east-1"
  key      = "key-1"
  secret   = "secret-1"
}

resource "coolify_s3_storage" "second" {
  name     = "second-storage"
  endpoint = "https://s3.eu-west-1.amazonaws.com"
  bucket   = "bucket-2"
  region   = "eu-west-1"
  key      = "key-2"
  secret   = "secret-2"
}

data "coolify_s3_storages" "all" {
  depends_on = [coolify_s3_storage.first, coolify_s3_storage.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_s3_storages.all", "s3_storages.#", "2"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_s3_storage" "first" {
  name     = "first-storage"
  endpoint = "https://s3.us-east-1.amazonaws.com"
  bucket   = "bucket-1"
  region   = "us-east-1"
  key      = "key-1"
  secret   = "secret-1"
}

resource "coolify_s3_storage" "second" {
  name     = "second-storage"
  endpoint = "https://s3.eu-west-1.amazonaws.com"
  bucket   = "bucket-2"
  region   = "eu-west-1"
  key      = "key-2"
  secret   = "secret-2"
}

data "coolify_s3_storages" "filtered" {
  depends_on = [coolify_s3_storage.first, coolify_s3_storage.second]
  filter {
    name   = "region"
    values = ["eu-west-1"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_s3_storages.filtered", "s3_storages.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.filtered", "s3_storages.0.name", "second-storage"),
				),
			},
		},
	})
}

func TestS3StorageListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_s3_storages" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing S3 storages`),
			},
		},
	})
}
