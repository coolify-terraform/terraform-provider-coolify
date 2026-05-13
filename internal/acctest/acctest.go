package acctest

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// WithVersionEndpoint wraps an http.Handler to also respond to
// GET /api/v1/version, which the provider calls during Configure
// to validate the connection.
func WithVersionEndpoint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/version" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`v4.0.0-test`))
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
	return fmt.Sprintf("%s-%d", prefix, rand.IntN(999999))
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

// CheckResourceDisappears returns a TestCheckFunc that deletes a resource
// out-of-band via the mock API. Use in Disappears tests to simulate external
// deletion. The deletePath should be the API path prefix (e.g., "/api/v1/servers/").
// The resource's "uuid" attribute is appended to the path.
//
//	Check: resource.ComposeAggregateTestCheckFunc(
//	    resource.TestCheckResourceAttrSet("coolify_server.test", "uuid"),
//	    acctest.CheckResourceDisappears(srv.URL, "coolify_server.test", "/api/v1/servers/"),
//	),
//	ExpectNonEmptyPlan: true,
func CheckResourceDisappears(serverURL, resourceAddr, apiPathPrefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		uuid := rs.Primary.Attributes["uuid"]
		if uuid == "" {
			return fmt.Errorf("resource %s has no uuid attribute", resourceAddr)
		}
		req, err := http.NewRequest(http.MethodDelete, serverURL+apiPathPrefix+uuid, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		return nil
	}
}

// CheckDestroy returns a TestCheckFunc that verifies a resource no longer
// exists via the mock API after the test completes. The apiPathPrefix is the
// GET endpoint prefix (e.g., "/api/v1/servers/").
func CheckDestroy(serverURL, resourceType, apiPathPrefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			uuid := rs.Primary.Attributes["uuid"]
			if uuid == "" {
				continue
			}
			req, err := http.NewRequest(http.MethodGet, serverURL+apiPathPrefix+uuid, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("error checking destroy for %s/%s: %w", resourceType, uuid, err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("%s %s still exists (status %d)", resourceType, uuid, resp.StatusCode)
			}
		}
		return nil
	}
}

// AccCheckDestroy returns a TestCheckFunc that verifies a resource no longer
// exists via the real Coolify API after the test completes. Unlike CheckDestroy
// (for mock servers), this includes Bearer token authentication.
func AccCheckDestroy(resourceType, apiPathPrefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		endpoint := os.Getenv("COOLIFY_ENDPOINT")
		token := os.Getenv("COOLIFY_TOKEN")
		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			uuid := rs.Primary.Attributes["uuid"]
			if uuid == "" {
				continue
			}
			req, err := http.NewRequest(http.MethodGet, endpoint+apiPathPrefix+uuid, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("error checking destroy for %s/%s: %w", resourceType, uuid, err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("%s %s still exists (status %d)", resourceType, uuid, resp.StatusCode)
			}
		}
		return nil
	}
}

// AccCheckNestedDestroy verifies a nested resource no longer exists by
// listing the parent's children and checking the child UUID is absent.
// parentAttr is the state attribute holding the parent UUID (e.g.,
// "application_uuid"). listPath is a format string with one %s for the
// parent UUID (e.g., "/api/v1/applications/%s/envs").
func AccCheckNestedDestroy(resourceType, parentAttr, listPath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			uuid := rs.Primary.Attributes["uuid"]
			parentUUID := rs.Primary.Attributes[parentAttr]
			if uuid == "" || parentUUID == "" {
				continue
			}
			if err := checkNestedResourceGone(parentUUID, uuid, resourceType, listPath); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkNestedResourceGone(parentUUID, uuid, resourceType, listPath string) error {
	endpoint := os.Getenv("COOLIFY_ENDPOINT")
	token := os.Getenv("COOLIFY_TOKEN")
	url := fmt.Sprintf("%s"+listPath, endpoint, parentUUID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error checking destroy for %s/%s: %w", resourceType, uuid, err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if strings.Contains(string(body), uuid) {
		return fmt.Errorf("%s %s still exists in parent %s", resourceType, uuid, parentUUID)
	}
	return nil
}

// ImportStateIDFunc returns an ImportStateIdFunc that reads a given attribute
// from the resource state. Use for resources whose import ID is stored in
// an attribute other than "id" (e.g., "uuid").
func ImportStateIDFunc(resourceAddr, attrName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		v := rs.Primary.Attributes[attrName]
		if v == "" {
			return "", fmt.Errorf("attribute %s is empty on %s", attrName, resourceAddr)
		}
		return v, nil
	}
}

// AccTestClient returns a Coolify API client configured from environment
// variables. Skips the test if COOLIFY_ENDPOINT or COOLIFY_TOKEN are not set.
func AccTestClient(t *testing.T) *client.Client {
	t.Helper()
	TestAccPreCheck(t)
	return client.New(os.Getenv("COOLIFY_ENDPOINT"), os.Getenv("COOLIFY_TOKEN"))
}

// AccTestServerUUID returns the UUID of a usable server from the Coolify
// instance. Checks COOLIFY_SERVER_UUID first, then queries the API for the
// first available server. Skips the test if no server is available.
func AccTestServerUUID(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("COOLIFY_SERVER_UUID"); v != "" {
		return v
	}
	c := AccTestClient(t)
	servers, err := c.ListServers(context.Background())
	if err != nil {
		t.Skipf("Could not list servers: %v", err)
	}
	for _, s := range servers {
		if s.UUID != "" {
			return s.UUID
		}
	}
	t.Skip("No servers available for acceptance tests")
	return ""
}

// AccTestDatabaseConfig returns a Terraform config for an acceptance test of a
// database resource, including a project dependency. The resourceType should be
// the full Terraform resource type (e.g. "coolify_postgresql_database").
func AccTestDatabaseConfig(resourceType, name, serverUUID, extra string) string {
	return ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" { name = %[1]q }
resource %[4]q "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
  %[3]s
}
`, name, serverUUID, extra, resourceType)
}

// AccTestSkipIfNoTFAcc skips the test if TF_ACC is not set.
func AccTestSkipIfNoTFAcc(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping acceptance test")
	}
}

// GenerateTestRSAKey generates a 2048-bit RSA private key in PEM format
// for use in acceptance tests. Fails the test on error.
func GenerateTestRSAKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %s", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}

// AccTestDockerfileAppConfig returns a Terraform config for an acceptance test
// of a Dockerfile application resource, including a project dependency. The
// extra parameter allows injecting additional HCL attributes.
func AccTestDockerfileAppConfig(name, serverUUID, extra string) string {
	return ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_dockerfile_application" "test" {
  project_uuid = coolify_project.test.uuid
  server_uuid  = %[2]q
  name         = %[1]q
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
  %[3]s
}
`, name, serverUUID, extra)
}
