package applicationpreview_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
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
	var deleted atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440041/previews/99", func(w http.ResponseWriter, _ *http.Request) {
		deleted.Store(true)
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
	if !deleted.Load() {
		t.Error("expected DELETE to be called on destroy")
	}
}

func TestApplicationPreviewResource_DeleteNotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440042/previews/7", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Preview not found."}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440042"
					pull_request_id  = 7
				`),
			},
			acctest.DestroyRemoveResourceStep(srv.URL),
		},
	})
}

func TestApplicationPreviewResource_DeleteError(t *testing.T) {
	t.Parallel()
	var gate acctest.DeleteOnceFailGate
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440043/previews/8", gate.Wrap(
		http.StatusOK,
		http.StatusInternalServerError,
		`{"message":"internal server error"}`,
	))
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440043"
					pull_request_id  = 8
				`),
			},
			acctest.DestroyExpectErrorStep(srv.URL, regexp.MustCompile(`Error deleting preview deployment`), &gate),
		},
	})
}
