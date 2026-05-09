package s3storage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestS3StoragesDataSource(t *testing.T) {
	storages := []client.S3Storage{
		{
			ID:          1,
			UUID:        "s3-list-uuid-1",
			Name:        "s3-alpha",
			Description: "First S3 storage",
			Endpoint:    "https://s3.us-east-1.amazonaws.com",
			Bucket:      "bucket-alpha",
			Region:      "us-east-1",
			AccessKey:   "AKIA1",
			SecretKey:   "secret1",
		},
		{
			ID:          2,
			UUID:        "s3-list-uuid-2",
			Name:        "s3-beta",
			Description: "Second S3 storage",
			Endpoint:    "https://s3.eu-west-1.amazonaws.com",
			Bucket:      "bucket-beta",
			Region:      "eu-west-1",
			AccessKey:   "AKIA2",
			SecretKey:   "secret2",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/storages" {
			json.NewEncoder(w).Encode(storages)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_s3_storages" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.0.uuid", "s3-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.0.name", "s3-alpha"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.0.description", "First S3 storage"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.0.endpoint", "https://s3.us-east-1.amazonaws.com"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.0.bucket", "bucket-alpha"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.0.region", "us-east-1"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.1.uuid", "s3-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.1.name", "s3-beta"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.1.endpoint", "https://s3.eu-west-1.amazonaws.com"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.1.bucket", "bucket-beta"),
					resource.TestCheckResourceAttr("data.coolify_s3_storages.test", "storages.1.region", "eu-west-1"),
				),
			},
		},
	})
}
