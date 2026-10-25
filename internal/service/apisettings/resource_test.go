package apisettings_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func apiSettingsMux(apiEnabled, mcpEnabled *atomic.Bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		apiEnabled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	mux.HandleFunc("POST /api/v1/disable", func(w http.ResponseWriter, _ *http.Request) {
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

func TestAPISettingsResource_MCPEnableError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	// MCP enable returns 403 (non-root token).
	mux.HandleFunc("POST /api/v1/mcp/enable", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"You are not allowed to perform this action."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					mcp_enabled = true
				`),
				ExpectError: regexp.MustCompile(`(?i)error configuring MCP`),
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

// ---------------------------------------------------------------------------
// TestAPISettingsResource_CreateAPIError
// ---------------------------------------------------------------------------

func TestAPISettingsResource_DestroyRestoreWarning(t *testing.T) {
	t.Parallel()
	var enableCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		if enableCalls.Add(1) == 1 {
			http.Error(w, `{"message":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API enabled."}`))
	})
	mux.HandleFunc("POST /api/v1/disable", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API disabled."}`))
	})
	mux.HandleFunc("POST /api/v1/mcp/disable", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"MCP server disabled."}`))
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
			},
			acctest.DestroyRemoveResourceStep(srv.URL),
		},
	})
	if enableCalls.Load() < 1 {
		t.Fatalf("expected enable API call on destroy restore, got %d", enableCalls.Load())
	}
}

// serveAPISettings pins the Coolify version the provider reads at
// Configure. getStatus 0 leaves GET /api/v1/enable unregistered. Go's
// ServeMux then answers 405, because POST /api/v1/enable is registered.
// A probe test that needs a real 404 must pass http.StatusNotFound.
func serveAPISettings(t *testing.T, version string, getStatus int) (*httptest.Server, *apiRouteObs) {
	t.Helper()
	obs := &apiRouteObs{}
	var apiEnabled, mcpEnabled atomic.Bool
	mux := apiSettingsMux(&apiEnabled, &mcpEnabled)
	if getStatus != 0 {
		mux.HandleFunc("GET /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(getStatus)
			_, _ = w.Write([]byte(`{"message":"post required"}`))
		})
	}
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/enable":
			obs.enablePosts.Add(1)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/disable":
			obs.disablePosts.Add(1)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/enable":
			obs.getEnable.Add(1)
		}
		mux.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(wrapped, version))
	t.Cleanup(srv.Close)
	return srv, obs
}

type apiRouteObs struct {
	enablePosts  atomic.Int32
	disablePosts atomic.Int32
	getEnable    atomic.Int32
}

func TestAPISettingsResource_Coolify447PostsEnable(t *testing.T) {
	t.Parallel()
	srv, obs := serveAPISettings(t, "4.4.7", 0)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
				enabled = true
			`),
			Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
		}},
	})
	// Create posts /enable, and destroy posts it again to restore the API.
	if got := obs.enablePosts.Load(); got != 2 {
		t.Fatalf("POST /api/v1/enable = %d, want 2", got)
	}
	if obs.getEnable.Load() != 0 {
		t.Fatal("honest 4.4.7 must not probe GET /api/v1/enable")
	}
}

func TestAPISettingsResource_Coolify45EnabledTrueSkipsEnable(t *testing.T) {
	t.Parallel()
	srv, obs := serveAPISettings(t, "4.5.0", 0)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
				enabled = true
			`),
			Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
		}},
	})
	// Destroy of an already-on API must not post /enable either.
	if got := obs.enablePosts.Load(); got != 0 {
		t.Fatalf("POST /api/v1/enable = %d, want 0", got)
	}
}

func TestAPISettingsResource_Coolify45DisableStillPosts(t *testing.T) {
	t.Parallel()
	srv, obs := serveAPISettings(t, "4.5.1", 0)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
				enabled = false
			`),
			Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "false"),
		}},
	})
	if obs.enablePosts.Load() != 0 {
		t.Fatalf("POST /api/v1/enable = %d, want 0", obs.enablePosts.Load())
	}
	if got := obs.disablePosts.Load(); got != 1 {
		t.Fatalf("POST /api/v1/disable = %d, want 1", got)
	}
}

func TestAPISettingsResource_Coolify45ReenableErrors(t *testing.T) {
	t.Parallel()
	srv, obs := serveAPISettings(t, "4.5.0", 0)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled = false
				`),
				Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "false"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled = true
				`),
				ExpectError: regexp.MustCompile(`Coolify 4\.5\.0 removed POST /api/v1/enable[\s\S]*remove this resource from state`),
			},
		},
	})
	if obs.enablePosts.Load() != 0 {
		t.Fatalf("POST /api/v1/enable = %d, want 0", obs.enablePosts.Load())
	}
	if got := obs.disablePosts.Load(); got != 1 {
		t.Fatalf("POST /api/v1/disable = %d, want 1", got)
	}
}

func TestAPISettingsResource_EdgeProbe405PostsEnable(t *testing.T) {
	t.Parallel()
	srv, obs := serveAPISettings(t, "4.3.0", http.StatusMethodNotAllowed)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
				enabled = true
			`),
			Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
		}},
	})
	if obs.getEnable.Load() < 1 {
		t.Fatal("expected GET /api/v1/enable probe")
	}
	if got := obs.enablePosts.Load(); got != 2 {
		t.Fatalf("POST /api/v1/enable = %d, want 2", got)
	}
}

func TestAPISettingsResource_EdgeProbe404SkipsEnable(t *testing.T) {
	t.Parallel()
	srv, obs := serveAPISettings(t, "4.3.0", http.StatusNotFound)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
				enabled = true
			`),
			Check: resource.TestCheckResourceAttr("coolify_api_settings.test", "enabled", "true"),
		}},
	})
	if obs.getEnable.Load() < 1 {
		t.Fatal("expected GET /api/v1/enable probe")
	}
	if got := obs.enablePosts.Load(); got != 0 {
		t.Fatalf("POST /api/v1/enable = %d, want 0", got)
	}
}

func TestAPISettingsResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/enable", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"internal server error"}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_api_settings", "test", `
					enabled     = true
					mcp_enabled = false
				`),
				ExpectError: regexp.MustCompile(`Error configuring API settings`),
			},
		},
	})
}
