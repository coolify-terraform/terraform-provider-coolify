package apisettings_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func apiSettingsMux(apiEnabled, mcpEnabled *atomic.Bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		apiEnabled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	mux.HandleFunc("GET /api/v1/disable", func(w http.ResponseWriter, _ *http.Request) {
		apiEnabled.Store(false)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API disabled."}`))
	})
	mux.HandleFunc("POST /api/v1/mcp/enable", func(w http.ResponseWriter, _ *http.Request) {
		mcpEnabled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"MCP server enabled."}`))
	})
	mux.HandleFunc("POST /api/v1/mcp/disable", func(w http.ResponseWriter, _ *http.Request) {
		mcpEnabled.Store(false)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"MCP server disabled."}`))
	})
	return mux
}

func TestAPISettingsResource_Enable(t *testing.T) {
	t.Parallel()
	var apiEnabled, mcpEnabled atomic.Bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(apiSettingsMux(&apiEnabled, &mcpEnabled)))
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
					resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "false"),
				),
			},
		},
	})
	// After destroy, the API should be re-enabled and MCP disabled.
	if !apiEnabled.Load() {
		t.Error("expected API to be re-enabled after destroy")
	}
}

func TestAPISettingsResource_Update(t *testing.T) {
	t.Parallel()
	var apiEnabled, mcpEnabled atomic.Bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(apiSettingsMux(&apiEnabled, &mcpEnabled)))
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
	var apiEnabled, mcpEnabled atomic.Bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(apiSettingsMux(&apiEnabled, &mcpEnabled)))
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

func TestAPISettingsResource_MCPEnable(t *testing.T) {
	t.Parallel()
	var apiEnabled, mcpEnabled atomic.Bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(apiSettingsMux(&apiEnabled, &mcpEnabled)))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					mcp_enabled = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "true"),
				),
			},
		},
	})
	if !apiEnabled.Load() {
		t.Error("expected API to be re-enabled after destroy")
	}
	if mcpEnabled.Load() {
		t.Error("expected MCP to be disabled after destroy")
	}
}

func TestAPISettingsResource_MCPUpdate(t *testing.T) {
	t.Parallel()
	var apiEnabled, mcpEnabled atomic.Bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(apiSettingsMux(&apiEnabled, &mcpEnabled)))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					mcp_enabled = false
				`),
				Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "false"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					mcp_enabled = true
				`),
				Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "true"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					mcp_enabled = false
				`),
				Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "false"),
			},
		},
	})
}

func TestAPISettingsResource_BothSettings(t *testing.T) {
	t.Parallel()
	var apiEnabled, mcpEnabled atomic.Bool
	srv := httptest.NewServer(acctest.WithVersionEndpoint(apiSettingsMux(&apiEnabled, &mcpEnabled)))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled     = true
					mcp_enabled = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_api_settings.test", "mcp_enabled", "true"),
				),
			},
		},
	})
}
