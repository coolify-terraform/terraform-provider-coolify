package cloudtokenvalidate_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudTokenValidateResource_Success(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/cloud-tokens/550e8400-e29b-41d4-a716-446655440030/validate", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Token is valid."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_cloud_token_validate", "test", `
					cloud_token_uuid = "550e8400-e29b-41d4-a716-446655440030"
				`),
				Check: resource.TestCheckResourceAttr("coolify_cloud_token_validate.test", "cloud_token_uuid", "550e8400-e29b-41d4-a716-446655440030"),
			},
		},
	})
}

func TestCloudTokenValidateResource_Failure(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/cloud-tokens/550e8400-e29b-41d4-a716-446655440031/validate", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Invalid token."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_cloud_token_validate", "test", `
					cloud_token_uuid = "550e8400-e29b-41d4-a716-446655440031"
				`),
				ExpectError: regexp.MustCompile(`(?i)cloud token validation failed`),
			},
		},
	})
}

func TestCloudTokenValidateResource_Triggers(t *testing.T) {
	t.Parallel()
	var callCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/cloud-tokens/550e8400-e29b-41d4-a716-446655440032/validate", func(w http.ResponseWriter, _ *http.Request) {
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
				Config: acctest.TestResourceConfig(srv.URL, "coolify_cloud_token_validate", "test", `
					cloud_token_uuid = "550e8400-e29b-41d4-a716-446655440032"
					triggers = { run = "1" }
				`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_cloud_token_validate", "test", `
					cloud_token_uuid = "550e8400-e29b-41d4-a716-446655440032"
					triggers = { run = "2" }
				`),
			},
		},
	})
	if callCount.Load() < 2 {
		t.Errorf("expected at least 2 validation calls, got %d", callCount.Load())
	}
}
