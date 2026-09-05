package applicationpreview_test

import (
	"encoding/json"
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

func TestApplicationPreviewResource_CreateSendsDomains(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440044/previews/11", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440044/previews/11", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440044"
					pull_request_id  = 11
					domains          = "https://pr.example.com"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "domains", "https://pr.example.com"),
				),
			},
		},
	})
	if gotBody["domains"] != "https://pr.example.com" {
		t.Fatalf("PATCH body domains = %#v", gotBody["domains"])
	}
	if _, ok := gotBody["docker_compose_domains"]; ok {
		t.Fatalf("PATCH body must omit docker_compose_domains, got %#v", gotBody["docker_compose_domains"])
	}
}

func TestApplicationPreviewResource_CreateDomainsTooOld(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440045"
					pull_request_id  = 12
					domains          = "https://pr.example.com"
				`),
				ExpectError: regexp.MustCompile(`preview domain updates require\s+Coolify >= v4\.3\.15`),
			},
		},
	})
}

func TestApplicationPreviewResource_UpdateSendsDomains(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440046/previews/13", func(w http.ResponseWriter, r *http.Request) {
		patchCount.Add(1)
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440046/previews/13", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440046"
					pull_request_id  = 13
					domains          = "https://pr-a.example.com"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "domains", "https://pr-a.example.com"),
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440046"
					pull_request_id  = 13
					domains          = "https://pr-b.example.com"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "domains", "https://pr-b.example.com"),
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440046"
					pull_request_id  = 13
				`),
			},
		},
	})
	if gotBody["domains"] != "https://pr-b.example.com" {
		t.Fatalf("last PATCH body domains = %#v", gotBody["domains"])
	}
	if got := patchCount.Load(); got != 2 {
		t.Fatalf("PATCH count = %d, want 2 (create + update, not clear)", got)
	}
}

func TestApplicationPreviewResource_CreateSendsDockerComposeDomains(t *testing.T) {
	t.Parallel()
	var gotBody map[string]json.RawMessage
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440047/previews/14", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440047/previews/14", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	const composeJSON = `[{"name":"web","domain":"https://pr.example.com"}]`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid       = "550e8400-e29b-41d4-a716-446655440047"
					pull_request_id        = 14
					docker_compose_domains = "[{\"name\":\"web\",\"domain\":\"https://pr.example.com\"}]"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "docker_compose_domains", composeJSON),
				),
			},
		},
	})
	if string(gotBody["docker_compose_domains"]) != composeJSON {
		t.Fatalf("PATCH docker_compose_domains = %s, want %s", gotBody["docker_compose_domains"], composeJSON)
	}
	if _, ok := gotBody["domains"]; ok {
		t.Fatalf("PATCH body must omit domains, got %s", gotBody["domains"])
	}
}

func TestApplicationPreviewResource_CreateForceOnlySkipsPatch(t *testing.T) {
	t.Parallel()
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440054/previews/21", func(w http.ResponseWriter, _ *http.Request) {
		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440054/previews/21", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid      = "550e8400-e29b-41d4-a716-446655440054"
					pull_request_id       = 21
					force_domain_override = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "force_domain_override", "true"),
				),
			},
		},
	})
	if got := patchCount.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0 (force-only is not a domain write)", got)
	}
}

func TestApplicationPreviewResource_CreateEmptyComposeSkipsPatch(t *testing.T) {
	t.Parallel()
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440055/previews/22", func(w http.ResponseWriter, _ *http.Request) {
		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440055/previews/22", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid       = "550e8400-e29b-41d4-a716-446655440055"
					pull_request_id        = 22
					docker_compose_domains = ""
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "docker_compose_domains", ""),
				),
			},
		},
	})
	if got := patchCount.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0 (empty compose is not a domain write)", got)
	}
}

func TestApplicationPreviewResource_CreateDomainsAndComposeConflict(t *testing.T) {
	t.Parallel()
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440056/previews/23", func(w http.ResponseWriter, _ *http.Request) {
		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid       = "550e8400-e29b-41d4-a716-446655440056"
					pull_request_id        = 23
					domains                = "https://pr.example.com"
					docker_compose_domains = "[{\"name\":\"web\",\"domain\":\"https://pr.example.com\"}]"
				`),
				ExpectError: regexp.MustCompile(`Attribute "domains" cannot be specified when "docker_compose_domains" is\s+specified`),
			},
		},
	})
	if got := patchCount.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0 (conflict rejected at plan)", got)
	}
}

func TestApplicationPreviewResource_CreateSendsForceDomainOverride(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440048/previews/15", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440048/previews/15", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid      = "550e8400-e29b-41d4-a716-446655440048"
					pull_request_id       = 15
					domains               = "https://pr.example.com"
					force_domain_override = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "force_domain_override", "true"),
				),
			},
		},
	})
	if gotBody["force_domain_override"] != true {
		t.Fatalf("PATCH force_domain_override = %#v, want true", gotBody["force_domain_override"])
	}
}

func TestApplicationPreviewResource_CreateDockerComposeDomainsObjectRejected(t *testing.T) {
	t.Parallel()
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440052/previews/19", func(w http.ResponseWriter, r *http.Request) {
		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid       = "550e8400-e29b-41d4-a716-446655440052"
					pull_request_id        = 19
					docker_compose_domains = "{\"web\":{\"domain\":\"https://pr.example.com\"}}"
				`),
				ExpectError: regexp.MustCompile(`docker_compose_domains must be a\s+JSON array`),
			},
		},
	})
	if got := patchCount.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0 (object form rejected before PATCH)", got)
	}
}

func TestApplicationPreviewResource_CreateInvalidDomains(t *testing.T) {
	t.Parallel()
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440053/previews/20", func(w http.ResponseWriter, _ *http.Request) {
		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440053"
					pull_request_id  = 20
					domains          = "not-a-url"
				`),
				ExpectError: regexp.MustCompile(`must be empty, or comma-separated http:// or https:// URLs`),
			},
		},
	})
	if got := patchCount.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0 (invalid domains rejected at plan)", got)
	}
}

func TestApplicationPreviewResource_CreateInvalidDockerComposeDomains(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440049/previews/16", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid       = "550e8400-e29b-41d4-a716-446655440049"
					pull_request_id        = 16
					docker_compose_domains = "not-json"
				`),
				ExpectError: regexp.MustCompile(`docker_compose_domains must be a\s+JSON array`),
			},
		},
	})
}

func TestApplicationPreviewResource_CreateDockerComposeDomainsNonObjectRejected(t *testing.T) {
	t.Parallel()
	var patchCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440059/previews/26", func(w http.ResponseWriter, _ *http.Request) {
		patchCount.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid       = "550e8400-e29b-41d4-a716-446655440059"
					pull_request_id        = 26
					docker_compose_domains = "[1]"
				`),
				ExpectError: regexp.MustCompile(`docker_compose_domains must be a\s+JSON array`),
			},
		},
	})
	if got := patchCount.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0 (non-object array rejected at plan)", got)
	}
}

func TestApplicationPreviewResource_CreatePatch404(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440057/previews/24", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["domains"] != "https://pr.example.com" {
			http.Error(w, "unexpected domains", http.StatusBadRequest)
			return
		}
		http.Error(w, `{"message":"Preview not found."}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440057"
					pull_request_id  = 24
					domains          = "https://pr.example.com"
				`),
				ExpectError: regexp.MustCompile(`Error updating preview domains`),
			},
		},
	})
}

func TestApplicationPreviewResource_CreatePatch409(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440058/previews/25", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["domains"] != "https://pr.example.com" {
			http.Error(w, "unexpected domains", http.StatusBadRequest)
			return
		}
		http.Error(w, `{"message":"Domain conflict.","warnings":["https://pr.example.com already in use"],"conflicts":["https://pr.example.com"]}`, http.StatusConflict)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440058"
					pull_request_id  = 25
					domains          = "https://pr.example.com"
				`),
				ExpectError: regexp.MustCompile(`Error updating preview domains`),
			},
		},
	})
}

func TestApplicationPreviewResource_CreatePatch500(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440050/previews/17", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["domains"] != "https://pr.example.com" {
			http.Error(w, "unexpected domains", http.StatusBadRequest)
			return
		}
		http.Error(w, `{"message":"internal server error"}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440050"
					pull_request_id  = 17
					domains          = "https://pr.example.com"
				`),
				ExpectError: regexp.MustCompile(`Error updating preview domains`),
			},
		},
	})
}

func TestApplicationPreviewResource_UpdateNoDomainWriteOnOldCoolify(t *testing.T) {
	t.Parallel()
	var patched atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440051/previews/18", func(w http.ResponseWriter, _ *http.Request) {
		patched.Store(true)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/550e8400-e29b-41d4-a716-446655440051/previews/18", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid = "550e8400-e29b-41d4-a716-446655440051"
					pull_request_id  = 18
				`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application_preview", "test", `
					application_uuid      = "550e8400-e29b-41d4-a716-446655440051"
					pull_request_id       = 18
					force_domain_override = false
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_preview.test", "force_domain_override", "false"),
				),
			},
		},
	})
	if patched.Load() {
		t.Fatal("expected no PATCH when Update has no domain writes")
	}
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
