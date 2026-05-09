package acctest

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// WithVersionEndpoint wraps an http.Handler to also respond to
// GET /api/v1/version, which the provider calls during Configure
// to validate the connection.
func WithVersionEndpoint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/version" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`"4.0.0-test"`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TestAccPreCheck validates that required environment variables are set.
func TestAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("COOLIFY_ENDPOINT"); v == "" {
		t.Fatal("COOLIFY_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("COOLIFY_TOKEN"); v == "" {
		t.Fatal("COOLIFY_TOKEN must be set for acceptance tests")
	}
}

// RandomWithPrefix generates a random string with the given prefix.
func RandomWithPrefix(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(999999))
}

// TestProtoV6ProviderFactories returns provider factories for unit tests.
func TestProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// ConfigProviderBlock returns a Terraform HCL provider configuration block
// using environment variables.
func ConfigProviderBlock() string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token     = %q
}
`, os.Getenv("COOLIFY_ENDPOINT"), os.Getenv("COOLIFY_TOKEN"))
}

// ProviderBlockForURL returns a provider block configured for a mock server URL.
func ProviderBlockForURL(serverURL string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}
`, serverURL)
}

// TestResourceConfig returns a Terraform config with a provider block and a single resource.
// This is the standard bootstrap helper for unit tests:
//
//	acctest.TestResourceConfig(srv.URL, "coolify_postgresql_database", "test", `
//	  project_uuid = "proj-uuid-1"
//	  server_uuid  = "srv-uuid-1"
//	`)
func TestResourceConfig(endpoint, resourceType, resourceName, attrs string) string {
	return fmt.Sprintf(`%s
resource "%s" "%s" {
  %s
}
`, ProviderBlockForURL(endpoint), resourceType, resourceName, attrs)
}

// TestDataSourceConfig returns a Terraform config with a provider block and a single data source.
func TestDataSourceConfig(endpoint, dataSourceType, dataSourceName, attrs string) string {
	return fmt.Sprintf(`%s
data "%s" "%s" {
  %s
}
`, ProviderBlockForURL(endpoint), dataSourceType, dataSourceName, attrs)
}

// RequireEnv skips the test if the given environment variable is not set.
// Returns the value if set.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("Environment variable %s not set, skipping", key)
	}
	return v
}
