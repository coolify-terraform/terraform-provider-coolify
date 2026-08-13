package s3storagevalidate_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestS3StorageValidateResource_Success(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/s3-storages/550e8400-e29b-41d4-a716-446655440040/validate", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Storage is usable."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage_validate", "test", `
					s3_storage_uuid = "550e8400-e29b-41d4-a716-446655440040"
				`),
				Check: resource.TestCheckResourceAttr("coolify_s3_storage_validate.test", "s3_storage_uuid", "550e8400-e29b-41d4-a716-446655440040"),
			},
		},
	})
}

func TestS3StorageValidateResource_InvalidUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage_validate", "test", `
					s3_storage_uuid = "not-a-valid-uuid"
				`),
				ExpectError: regexp.MustCompile(`(?i)uuid|invalid|must`),
			},
		},
	})
}

func TestS3StorageValidateResource_Failure(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/s3-storages/550e8400-e29b-41d4-a716-446655440041/validate", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Connection refused."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage_validate", "test", `
					s3_storage_uuid = "550e8400-e29b-41d4-a716-446655440041"
				`),
				ExpectError: regexp.MustCompile(`(?i)s3 storage validation failed`),
			},
		},
	})
}

func TestS3StorageValidateResource_Triggers(t *testing.T) {
	t.Parallel()
	var callCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/s3-storages/550e8400-e29b-41d4-a716-446655440042/validate", func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"OK"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage_validate", "test", `
					s3_storage_uuid = "550e8400-e29b-41d4-a716-446655440042"
					triggers = { run = "1" }
				`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage_validate", "test", `
					s3_storage_uuid = "550e8400-e29b-41d4-a716-446655440042"
					triggers = { run = "2" }
				`),
			},
		},
	})
	if callCount.Load() < 2 {
		t.Errorf("expected at least 2 validation calls, got %d", callCount.Load())
	}
}

func TestS3StorageValidateResource_DestroyNoOp(t *testing.T) {
	t.Parallel()
	var validateCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/s3-storages/550e8400-e29b-41d4-a716-446655440043/validate", func(w http.ResponseWriter, _ *http.Request) {
		validateCalls.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"OK"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage_validate", "test", `
					s3_storage_uuid = "550e8400-e29b-41d4-a716-446655440043"
				`),
			},
			acctest.DestroyRemoveResourceStep(srv.URL),
		},
	})
	if validateCalls.Load() != 1 {
		t.Fatalf("expected validation only on create, got %d calls", validateCalls.Load())
	}
}
