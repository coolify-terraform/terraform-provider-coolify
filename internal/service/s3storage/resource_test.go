package s3storage_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func newS3StorageMockServer() *httptest.Server {
	storages := make(map[string]*client.S3Storage)
	var mu sync.Mutex

	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/storages":
			var input client.CreateS3StorageInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			s := &client.S3Storage{
				ID:          1,
				UUID:        "s3-test-uuid-1",
				Name:        input.Name,
				Description: input.Description,
				Endpoint:    input.Endpoint,
				Bucket:      input.Bucket,
				Region:      input.Region,
				AccessKey:   input.AccessKey,
				SecretKey:   input.SecretKey,
			}
			storages[s.UUID] = s
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(s)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/storages":
			list := make([]client.S3Storage, 0, len(storages))
			for _, s := range storages {
				list = append(list, *s)
			}
			json.NewEncoder(w).Encode(list)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/storages/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/storages/")
			s, ok := storages[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(s)

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/storages/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/storages/")
			s, ok := storages[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var update client.UpdateS3StorageInput
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			if update.Name != nil {
				s.Name = *update.Name
			}
			if update.Description != nil {
				s.Description = *update.Description
			}
			if update.Endpoint != nil {
				s.Endpoint = *update.Endpoint
			}
			if update.Bucket != nil {
				s.Bucket = *update.Bucket
			}
			if update.Region != nil {
				s.Region = *update.Region
			}
			if update.AccessKey != nil {
				s.AccessKey = *update.AccessKey
			}
			if update.SecretKey != nil {
				s.SecretKey = *update.SecretKey
			}
			json.NewEncoder(w).Encode(s)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/storages/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/storages/")
			delete(storages, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})))
}

func TestS3StorageResource_Create(t *testing.T) {
	t.Parallel()
	srv := newS3StorageMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_s3_storage" "test" {
  name        = "my-s3"
  description = "Test S3 storage"
  endpoint    = "https://s3.amazonaws.com"
  bucket      = "my-bucket"
  region      = "us-east-1"
  access_key  = "AKIAIOSFODNN7EXAMPLE"
  secret_key  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "uuid", "s3-test-uuid-1"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", "my-s3"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "description", "Test S3 storage"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "https://s3.amazonaws.com"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "my-bucket"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "us-east-1"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_s3_storage" "test" {
  name        = "my-s3"
  description = "Test S3 storage"
  endpoint    = "https://s3.amazonaws.com"
  bucket      = "my-bucket"
  region      = "us-east-1"
  access_key  = "AKIAIOSFODNN7EXAMPLE"
  secret_key  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestS3StorageResource_Update(t *testing.T) {
	t.Parallel()
	srv := newS3StorageMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_s3_storage" "test" {
  name       = "my-s3"
  endpoint   = "https://s3.amazonaws.com"
  bucket     = "my-bucket"
  region     = "us-east-1"
  access_key = "AKIAIOSFODNN7EXAMPLE"
  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", "my-s3"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "https://s3.amazonaws.com"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_s3_storage" "test" {
  name        = "updated-s3"
  description = "Updated description"
  endpoint    = "https://s3.amazonaws.com"
  bucket      = "my-bucket"
  region      = "us-east-1"
  access_key  = "AKIAIOSFODNN7EXAMPLE"
  secret_key  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "uuid", "s3-test-uuid-1"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", "updated-s3"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestS3StorageResource_Import(t *testing.T) {
	t.Parallel()
	srv := newS3StorageMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_s3_storage" "test" {
  name       = "import-s3"
  endpoint   = "https://s3.amazonaws.com"
  bucket     = "my-bucket"
  region     = "us-east-1"
  access_key = "AKIAIOSFODNN7EXAMPLE"
  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "uuid", "s3-test-uuid-1"),
				),
			},
			{
				ResourceName:                         "coolify_s3_storage.test",
				ImportState:                          true,
				ImportStateId:                        "s3-test-uuid-1",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestS3StorageResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newS3StorageMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_s3_storage" "test" {
  name       = "disappearing-s3"
  endpoint   = "https://s3.amazonaws.com"
  bucket     = "my-bucket"
  region     = "us-east-1"
  access_key = "AKIAIOSFODNN7EXAMPLE"
  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_s3_storage.test", "uuid"),
					// Delete the storage out-of-band via the mock API.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_s3_storage.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/storages/"+uuid, nil)
						if err != nil {
							return err
						}
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestS3StorageResource_InvalidEndpoint(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_s3_storage" "test" {
  name       = "my-s3"
  endpoint   = "not-a-url"
  bucket     = "my-bucket"
  region     = "us-east-1"
  access_key = "AKIA123"
  secret_key = "secret"
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`must start with http:// or https://`),
			},
		},
	})
}
