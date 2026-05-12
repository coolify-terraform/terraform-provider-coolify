package s3storage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestS3StorageDataSource(t *testing.T) {
	t.Parallel()
	s := &client.S3Storage{
		ID:          1,
		UUID:        "cccc0005-0005-4000-8000-000000000001",
		Name:        "data-source-s3",
		Description: "An S3 storage for testing",
		Endpoint:    "https://s3.us-west-2.amazonaws.com",
		Bucket:      "test-bucket",
		Region:      "us-west-2",
		AccessKey:   "AKIAIOSFODNN7EXAMPLE",
		SecretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/storages/") {
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/storages/")
			if uuid == s.UUID {
				json.NewEncoder(w).Encode(s)
				return
			}
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_s3_storage" "test" {
  uuid = "cccc0005-0005-4000-8000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "uuid", "cccc0005-0005-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "name", "data-source-s3"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "description", "An S3 storage for testing"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "endpoint", "https://s3.us-west-2.amazonaws.com"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "bucket", "test-bucket"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "region", "us-west-2"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "access_key", "AKIAIOSFODNN7EXAMPLE"),
					resource.TestCheckResourceAttr("data.coolify_s3_storage.test", "secret_key", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				),
			},
		},
	})
}
