package s3storage_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestS3StorageResource_SFTPRejectsBucket(t *testing.T) {
	t.Parallel()
	var posts atomic.Int32
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"s3-1"}`))
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  bucket   = "b"
  key      = "user"
  secret   = "secret"
`),
			ExpectError: regexp.MustCompile(`bucket is only for s3 storage`),
		}},
	})
	if posts.Load() != 0 {
		t.Fatalf("sftp create with bucket sent %d POSTs", posts.Load())
	}
}

func TestS3StorageResource_OldCoolifyRejectsSFTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request %s %s on Coolify 4.6", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}), "4.6.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  key      = "user"
  secret   = "secret"
`),
			ExpectError: regexp.MustCompile(`Coolify version cannot set storage type`),
		}},
	})
}

func TestS3StorageResource_S3RequiresBucket(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/version" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "https://s3.example.com"
  key      = "access"
  secret   = "secret"
`),
			ExpectError: regexp.MustCompile(`bucket is required for s3 storage`),
		}},
	})
}
