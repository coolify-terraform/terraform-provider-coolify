package servervalidate_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServerValidateResource_Valid(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/550e8400-e29b-41d4-a716-446655440020/validate", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"valid":true,"message":"Server is reachable."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_server_validate", "test", `
					server_uuid = "550e8400-e29b-41d4-a716-446655440020"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_validate.test", "valid", "true"),
					resource.TestCheckResourceAttr("coolify_server_validate.test", "message", "Server is reachable."),
				),
			},
		},
	})
}

func TestServerValidateResource_Invalid(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/550e8400-e29b-41d4-a716-446655440021/validate", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"valid":false,"message":"SSH connection refused."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_server_validate", "test", `
					server_uuid = "550e8400-e29b-41d4-a716-446655440021"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_validate.test", "valid", "false"),
					resource.TestCheckResourceAttr("coolify_server_validate.test", "message", "SSH connection refused."),
				),
			},
		},
	})
}

func TestServerValidateResource_Triggers(t *testing.T) {
	t.Parallel()
	var callCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/550e8400-e29b-41d4-a716-446655440022/validate", func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)
		_, _ = w.Write([]byte(`{"valid":true,"message":"OK"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_server_validate", "test", `
					server_uuid = "550e8400-e29b-41d4-a716-446655440022"
					triggers = { run = "1" }
				`),
				Check: resource.TestCheckResourceAttr("coolify_server_validate.test", "valid", "true"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_server_validate", "test", `
					server_uuid = "550e8400-e29b-41d4-a716-446655440022"
					triggers = { run = "2" }
				`),
				Check: resource.TestCheckResourceAttr("coolify_server_validate.test", "valid", "true"),
			},
		},
	})
	if callCount.Load() < 2 {
		t.Errorf("expected at least 2 validation calls, got %d", callCount.Load())
	}
}
