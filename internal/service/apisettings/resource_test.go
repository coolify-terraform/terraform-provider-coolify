package apisettings_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAPISettingsResource_Enable(t *testing.T) {
	t.Parallel()
	var enabled atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		enabled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	mux.HandleFunc("GET /api/v1/disable", func(w http.ResponseWriter, _ *http.Request) {
		enabled.Store(false)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API disabled."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
				),
			},
		},
	})
	// After destroy, the API should be re-enabled.
	if !enabled.Load() {
		t.Error("expected API to be re-enabled after destroy")
	}
}

func TestAPISettingsResource_Update(t *testing.T) {
	t.Parallel()
	var enabled atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		enabled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	mux.HandleFunc("GET /api/v1/disable", func(w http.ResponseWriter, _ *http.Request) {
		enabled.Store(false)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API disabled."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled = false
				`),
				Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "false"),
			},
		},
	})
}

func TestAPISettingsResource_Disable(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	mux.HandleFunc("GET /api/v1/disable", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API disabled."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled = false
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "false"),
				),
			},
		},
	})
}
