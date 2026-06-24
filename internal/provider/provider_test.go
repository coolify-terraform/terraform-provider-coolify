package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProvider_MissingToken(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://localhost:8000"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Missing Coolify Token`),
			},
		},
	})
}

func TestProvider_MissingEndpoint(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  token = "some-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Missing Coolify Endpoint`),
			},
		},
	})
}

func TestProvider_UnreachableEndpoint(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://127.0.0.1:1"
  token    = "bad-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Unable to connect to Coolify`),
			},
		},
	})
}

func TestProvider_InvalidToken(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			http.Error(w, `{"message":"Unauthenticated."}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Unable to connect to Coolify`),
			},
		},
	})
}

func TestProvider_InvalidEndpointScheme(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "ftp://localhost:8000"
  token    = "test-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Invalid Coolify Endpoint`),
			},
		},
	})
}

func TestProvider_EnvVarPrecedence(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv().
	// Set env vars that should be overridden by the provider block.
	t.Setenv("COOLIFY_ENDPOINT", "http://127.0.0.1:1") // unreachable
	t.Setenv("COOLIFY_TOKEN", "env-token")

	// The provider block endpoint (mock server) overrides the env var.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("v4.1.0"))
	})
	mux.HandleFunc("GET /api/v1/teams/0", func(w http.ResponseWriter, r *http.Request) {
		// Verify the config-block token overrides the env var.
		auth := r.Header.Get("Authorization")
		if auth != "Bearer config-token" {
			http.Error(w, `{"message":"wrong token"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":0,"name":"Root Team"}`)
	})
	mux.HandleFunc("GET /api/v1/teams/0/members", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "config-token"
}

data "coolify_team" "test" {
  id = 0
}
`, srv.URL),
				Check: resource.TestCheckResourceAttr("data.coolify_team.test", "name", "Root Team"),
			},
		},
	})
}

func TestProvider_CloudflareAccessHeaders(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		// Verify CF Access headers are set on the health check call.
		cfID := r.Header.Get("CF-Access-Client-Id")
		cfSec := r.Header.Get("CF-Access-Client-Secret")
		if cfID != "test-cf-id" || cfSec != "test-cf-secret" {
			http.Error(w, "missing CF headers", http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("v4.1.0"))
	})
	mux.HandleFunc("GET /api/v1/teams/0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":0,"name":"CF Team"}`)
	})
	mux.HandleFunc("GET /api/v1/teams/0/members", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint                = %q
  token                   = "test-token"
  cf_access_client_id     = "test-cf-id"
  cf_access_client_secret = "test-cf-secret"
}

data "coolify_team" "test" {
  id = 0
}
`, srv.URL),
				Check: resource.TestCheckResourceAttr("data.coolify_team.test", "name", "CF Team"),
			},
		},
	})
}

func TestProvider_RetryConfiguration(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NewServeMux()))
	defer srv.Close()

	// Verify custom retry settings are accepted without error.
	// The data source call will 404 (no handler), but we only care that
	// provider Configure succeeds with the retry config.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint       = %q
  token          = "test-token"
  retry_max      = 1
  retry_min_wait = 1
  retry_max_wait = 5
}

data "coolify_team" "test" {
  id = 0
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`Error reading team`),
			},
		},
	})
}

func TestProvider_InvalidCACert(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "https://coolify.example.com"
  token    = "test-token"
  ca_cert  = "not-a-valid-pem"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Invalid CA Certificate`),
			},
		},
	})
}

func TestProvider_UnsupportedVersion(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			_, _ = w.Write([]byte("v3.9.0"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_team" "test" {
  id = 0
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`Unsupported Coolify version`),
			},
		},
	})
}

func TestProvider_UnreachableEndpointRedactsCredentials(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://user:pass@127.0.0.1:1"
  token    = "bad-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`(?s)Unable to connect to Coolify.*http://REDACTED:REDACTED@127\.0\.0\.1:1`),
			},
		},
	})
}

func TestProvider_EnvVarFallback(t *testing.T) {
	// Configure everything via env vars only (no provider block attributes).
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("v4.1.0"))
	})
	mux.HandleFunc("GET /api/v1/teams/0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":0,"name":"Env Team"}`)
	})
	mux.HandleFunc("GET /api/v1/teams/0/members", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "env-only-token")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {}

data "coolify_team" "test" {
  id = 0
}
`,
				Check: resource.TestCheckResourceAttr("data.coolify_team.test", "name", "Env Team"),
			},
		},
	})
}
