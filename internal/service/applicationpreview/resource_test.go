package applicationpreview_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationPreviewResource_Create(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440040/previews/42", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440040"
					pull_request_id  = 42
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "application_uuid", "550e8400-e29b-41d4-a716-446655440040"),
					resource.TestCheckResourceAttr("coolify_application_preview.test", "pull_request_id", "42"),
				),
			},
		},
	})
}

func TestApplicationPreviewResource_DeleteCalled(t *testing.T) {
	t.Parallel()
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440041/previews/99", func(w http.ResponseWriter, _ *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440041"
					pull_request_id  = 99
				`),
			},
		},
	})
	if !deleted {
		t.Error("expected DELETE to be called on destroy")
	}
}
