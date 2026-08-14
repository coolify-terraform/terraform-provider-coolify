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
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// DefaultTestCoolifyVersion is the version mock servers report by default.
//
// It is a 4.2.x version because the provider gates some writes on the connected
// version (see client.SupportsApplicationSettings): a mock reporting 4.1.x would
// silently put every unit test on the withholding path, and tests asserting that
// a settings field reaches the API would fail for reasons unrelated to what they
// cover. Use WithVersionEndpointVersion to exercise the older behaviour on
// purpose.
const DefaultTestCoolifyVersion = "v4.2.0-test"

// WithVersionEndpoint wraps an http.Handler to also respond to
// GET /api/v1/version, which the provider calls during Configure
// to validate the connection.
func WithVersionEndpoint(next http.Handler) http.Handler {
	return WithVersionEndpointVersion(next, DefaultTestCoolifyVersion)
}

// WithVersionEndpointVersion is WithVersionEndpoint with an explicit version,
// for tests that need to pin the connected Coolify to a particular release.
func WithVersionEndpointVersion(next http.Handler, version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/version" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(version))
			return
		}
		next.ServeHTTP(w, r)
	})
}

var (
	accConnectivityCheckOnce sync.Once
	accConnectivityCheckErr  error
	accServerDiscoveryOnce   sync.Once
	accServerDiscoveryUUID   string
	accServerDiscoveryErr    error
)

func resetAccTestCaches() {
	accConnectivityCheckOnce = sync.Once{}
	accConnectivityCheckErr = nil
	accServerDiscoveryOnce = sync.Once{}
	accServerDiscoveryUUID = ""
	accServerDiscoveryErr = nil
}

// TestAccPreCheck validates that required environment variables are set.
func TestAccPreCheck(t *testing.T) {
	t.Helper()
	endpoint := os.Getenv("COOLIFY_ENDPOINT")
	if endpoint == "" {
		t.Fatal("COOLIFY_ENDPOINT must be set for acceptance tests. Run 'make acc-preflight' after exporting your local Coolify credentials, or bootstrap with 'make acc-bootstrap'.")
	}
	token := os.Getenv("COOLIFY_TOKEN")
	if token == "" {
		t.Fatal("COOLIFY_TOKEN must be set for acceptance tests. Run 'make acc-preflight' after exporting your local Coolify credentials, or bootstrap with 'make acc-bootstrap'.")
	}

	accConnectivityCheckOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		c := client.New(endpoint, token)
		if _, err := c.GetVersion(ctx); err != nil {
			accConnectivityCheckErr = fmt.Errorf("checking Coolify API connectivity: %w. Run 'make acc-preflight' or 'make acc-bootstrap' to verify the local instance", err)
		}
	})
	if accConnectivityCheckErr != nil {
		t.Fatal(accConnectivityCheckErr)
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
//	acctest.TestResourceConfig(srv.URL, "coolify_database_postgresql", "test", `
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

// TestDataSourceConfig builds an HCL config block for a data source test.
// Usage:
//
//	acctest.TestDataSourceConfig(srv.URL, "coolify_project", "test", `
//	  uuid = "ds-uuid-0001"
//	`)
func TestDataSourceConfig(endpoint, dataSourceType, dataSourceName, attrs string) string {
	return fmt.Sprintf(`%s
data "%s" "%s" {
  %s
}
`, ProviderBlockForURL(endpoint), dataSourceType, dataSourceName, attrs)
}

// DeleteOnceFailGate fails the first DELETE request when armed, then succeeds on
// later calls so post-test destroy cleanup does not false-fail ExpectError steps.
type DeleteOnceFailGate struct {
	Armed atomic.Bool
	Calls atomic.Int32
}

// Wrap returns a DELETE handler that responds with failStatus/failBody on the
// first call while armed, and okStatus on subsequent calls.
func (g *DeleteOnceFailGate) Wrap(okStatus, failStatus int, failBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if g.Armed.Load() && g.Calls.Add(1) == 1 {
			http.Error(w, failBody, failStatus)
			return
		}
		w.WriteHeader(okStatus)
	}
}

// DestroyRemoveResourceStep removes all resources from config, triggering destroy.
func DestroyRemoveResourceStep(serverURL string) resource.TestStep {
	return resource.TestStep{
		Config:             ProviderBlockForURL(serverURL),
		ExpectNonEmptyPlan: false,
	}
}

// DestroyExpectErrorStep expects destroy to fail after arming the delete gate.
func DestroyExpectErrorStep(serverURL string, expectError *regexp.Regexp, gate *DeleteOnceFailGate) resource.TestStep {
	return resource.TestStep{
		PreConfig: func() {
			gate.Armed.Store(true)
		},
		Config:      ProviderBlockForURL(serverURL),
		ExpectError: expectError,
	}
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
		if resp.StatusCode >= 400 {
			return fmt.Errorf("DELETE %s%s%s returned status %d", serverURL, apiPathPrefix, uuid, resp.StatusCode)
		}
		return nil
	}
}

// CheckPathDisappears deletes an exact API path out-of-band. Use for resources
// whose remote identity is not a single uuid attribute (shared envs, attach
// resources). The next plan must be non-empty.
func CheckPathDisappears(serverURL, path string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		req, err := http.NewRequest(http.MethodDelete, serverURL+path, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("DELETE %s%s returned status %d", serverURL, path, resp.StatusCode)
		}
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

// discoverServerUUID resolves a server UUID from the Coolify API. When
// overrideUUID is non-empty, it must appear in the server list; otherwise the
// first visible server is used.
func discoverServerUUID(endpoint, token, overrideUUID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c := client.New(endpoint, token)
	servers, err := c.ListServers(ctx)
	if err != nil {
		return "", fmt.Errorf("listing servers: %w", err)
	}

	first := ""
	for _, s := range servers {
		if s.UUID == "" {
			continue
		}
		if overrideUUID != "" && s.UUID == overrideUUID {
			return s.UUID, nil
		}
		if first == "" {
			first = s.UUID
		}
	}

	if overrideUUID != "" {
		return "", fmt.Errorf("COOLIFY_SERVER_UUID %q was not returned by the API", overrideUUID)
	}
	if first == "" {
		return "", fmt.Errorf("no visible servers returned by the API")
	}
	return first, nil
}

// AccTestServerUUID returns the UUID of a usable server from the Coolify
// instance. If COOLIFY_SERVER_UUID is set, it must be visible to the current
// API token. Otherwise the first visible server is used. Skips the test if no
// server is available.
func AccTestServerUUID(t *testing.T) string {
	t.Helper()
	TestAccPreCheck(t)

	overrideUUID := os.Getenv("COOLIFY_SERVER_UUID")
	accServerDiscoveryOnce.Do(func() {
		accServerDiscoveryUUID, accServerDiscoveryErr = discoverServerUUID(
			os.Getenv("COOLIFY_ENDPOINT"),
			os.Getenv("COOLIFY_TOKEN"),
			overrideUUID,
		)
	})
	if accServerDiscoveryErr != nil {
		t.Skipf("No server fixture available for acceptance tests: %v. Set COOLIFY_SERVER_UUID explicitly, or run 'make acc-preflight' / 'make acc-bootstrap' to validate a local server.", accServerDiscoveryErr)
	}
	return accServerDiscoveryUUID
}

// AccTestSecondServerUUID returns a second Coolify server UUID for extra
// destination attach tests. Prefers COOLIFY_SECOND_SERVER_UUID when that
// server is visible and different from AccTestServerUUID. Otherwise picks
// the first listed server that is not the primary. Skips when only one
// server is available (same-server extra destinations are 422).
func AccTestSecondServerUUID(t *testing.T) string {
	t.Helper()
	primary := AccTestServerUUID(t)
	endpoint := os.Getenv("COOLIFY_ENDPOINT")
	token := os.Getenv("COOLIFY_TOKEN")
	override := os.Getenv("COOLIFY_SECOND_SERVER_UUID")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := client.New(endpoint, token)
	servers, err := c.ListServers(ctx)
	if err != nil {
		t.Skipf("listing servers for extra destination: %v", err)
	}

	if override != "" {
		if override == primary {
			t.Skipf("COOLIFY_SECOND_SERVER_UUID %q is the same as the primary server; extra destination attach requires a second server", override)
		}
		for _, s := range servers {
			if s.UUID == override {
				return s.UUID
			}
		}
		t.Skipf("COOLIFY_SECOND_SERVER_UUID %q was not returned by the API", override)
	}

	for _, s := range servers {
		if s.UUID != "" && s.UUID != primary {
			return s.UUID
		}
	}
	t.Skip("No second Coolify server available. Set COOLIFY_SECOND_SERVER_UUID or add another server. Same-server extra destinations are 422.")
	return ""
}

// AccTestDatabaseConfig returns a Terraform config for an acceptance test of a
// database resource, including a project dependency. The resourceType should be
// the full Terraform resource type (e.g. "coolify_database_postgresql").
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

// UUIDValidationError returns a regex matching the provider's UUID validator
// rejection message. Use with ExpectError in negative validation tests.
func UUIDValidationError() *regexp.Regexp {
	return regexp.MustCompile(`(?i)must be a valid UUID`)
}

// NotFoundError returns a regex matching common "not found" error messages
// from both the provider and the Coolify API, including diagnostics where
// wrapping introduces extra whitespace or newlines between the words.
func NotFoundError() *regexp.Regexp {
	return regexp.MustCompile(`(?i)not\s+found`)
}

// AccCheckResourceDisappears returns a TestCheckFunc that deletes a resource
// out-of-band via the real Coolify API. Use in acceptance Disappears tests to
// simulate external deletion. The apiDeletePath should be the API path prefix
// (e.g., "/api/v1/projects/"). The resource's "uuid" attribute is appended.
//
// After deleting, it polls the GET endpoint until the resource returns 404
// to handle Coolify's async deletion behavior.
func AccCheckResourceDisappears(resourceAddr, apiDeletePath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		uuid := rs.Primary.Attributes["uuid"]
		if uuid == "" {
			return fmt.Errorf("resource %s has no uuid attribute", resourceAddr)
		}
		endpoint := os.Getenv("COOLIFY_ENDPOINT")
		token := os.Getenv("COOLIFY_TOKEN")

		// Delete the resource
		req, err := http.NewRequest(http.MethodDelete, endpoint+apiDeletePath+uuid, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("deleting %s/%s: %w", resourceAddr, uuid, err)
		}
		_ = resp.Body.Close()

		// Poll until the resource is gone (Coolify deletes are async).
		// Applications can take 60-120s to fully remove on slow hosts.
		for range 120 {
			time.Sleep(1 * time.Second)
			getReq, err := http.NewRequest(http.MethodGet, endpoint+apiDeletePath+uuid, nil)
			if err != nil {
				return err
			}
			getReq.Header.Set("Authorization", "Bearer "+token)
			getResp, err := http.DefaultClient.Do(getReq)
			if err != nil {
				continue
			}
			_ = getResp.Body.Close()
			if getResp.StatusCode == http.StatusNotFound {
				return nil
			}
		}
		return fmt.Errorf("resource %s/%s still exists after 120s", resourceAddr, uuid)
	}
}

// requireTipAPIs reports whether tip-only Coolify surfaces must be present.
// Set COOLIFY_REQUIRE_TIP_APIS=1 on jobs that boot coollabsio/coolify:edge
// (PR Acceptance Tests + Coolify Nightly Acc tip-edge) so missing S3 storage,
// volume backup, or bootstrap S3 UUID fails the run instead of silently
// skipping features the version-support guide claims on tip.
func requireTipAPIs() bool {
	v := strings.TrimSpace(os.Getenv("COOLIFY_REQUIRE_TIP_APIS"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// accTestMissingFeature skips unless COOLIFY_REQUIRE_TIP_APIS is set, in which
// case it fails the test (tip must expose claimed APIs).
func accTestMissingFeature(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	if requireTipAPIs() {
		t.Fatalf("tip Coolify is missing a claimed feature (COOLIFY_REQUIRE_TIP_APIS=1): %s", msg)
	}
	t.Skip(msg)
}

// AccTestSkipIfNoVolumeBackupAPI skips when Coolify lacks VolumeBackupsController
// (PUT/DELETE .../storages/{uuid}/backups from coollabsio/coolify#10946).
//
// That API is on Coolify branch v4.x after the merge; it is not in git tag
// v4.2.0 or stable CDN latest. CI boots coollabsio/coolify:edge (v4.x tip).
// Locally set LATEST_IMAGE=edge (or a post-#10946 sha-... tag) before compose up.
//
// When COOLIFY_REQUIRE_TIP_APIS=1, a missing controller fails instead of skips.
//
// Probe strategy: PUT a schedule for non-existent parent/storage UUIDs.
// Present controller: named resource 404 (see isControllerResourceNotFound),
// 422 validation, or 401/403. Missing route: plain {"message":"Not found."}
// (Coolify catch-all), empty body, or HTML (floor 4.1.2 regression).
func AccTestSkipIfNoVolumeBackupAPI(t *testing.T) {
	t.Helper()
	TestAccPreCheck(t)

	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	token := os.Getenv("COOLIFY_TOKEN")
	// NanoID-shaped placeholders; do not need to exist.
	path := endpoint + "/api/v1/applications/tfaccprobe000000000000001/storages/tfaccprobe000000000000002/backups"
	req, err := http.NewRequest(http.MethodPut, path, strings.NewReader(`{"frequency":"daily"}`))
	if err != nil {
		t.Fatalf("building volume backup probe request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		accTestMissingFeature(t, "volume backup API probe failed (cannot reach Coolify): %v", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
	msg := strings.ToLower(string(body))

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		// Route is registered; auth/middleware ran.
		return
	case http.StatusUnprocessableEntity:
		// Validation ran on the volume backup controller.
		return
	case http.StatusNotFound:
		// Controller-style 404s (named resource) mean the route exists.
		// Plain "Not found." / empty / Laravel unmatched means no controller.
		if isControllerResourceNotFound(msg) {
			return
		}
		accTestMissingFeature(t, "Coolify instance has no VolumeBackupsController (HTTP %d). "+
			"Need image tip after coollabsio/coolify#10946 (CI uses coollabsio/coolify:edge; "+
			"local: set LATEST_IMAGE=edge in Coolify compose .env). Body: %s",
			resp.StatusCode, truncateForSkip(string(body), 200))
		return
	case http.StatusMethodNotAllowed:
		accTestMissingFeature(t, "Coolify instance has no volume backup PUT route (HTTP 405). "+
			"Need tip after coollabsio/coolify#10946 (coollabsio/coolify:edge)")
	default:
		// 200/201 should not happen for fake UUIDs; other codes still mean a handler ran.
		if resp.StatusCode >= 500 {
			accTestMissingFeature(t, "volume backup API probe returned HTTP %d (server error): %s",
				resp.StatusCode, truncateForSkip(string(body), 200))
		}
	}
}

func truncateForSkip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// isControllerResourceNotFound reports whether a lowercased 404 body looks like
// a Coolify controller response for a missing parent/resource (route present).
// Unmatched routes typically return plain "Not found." without naming a resource.
func isControllerResourceNotFound(msgLower string) bool {
	for _, s := range []string{
		"application not found",
		"resource not found",
		"storage not found",
		"database not found",
		"service not found",
		"server not found",
		"s3 storage not found",
		"destination not found",
		"backup not found",
	} {
		if strings.Contains(msgLower, s) {
			return true
		}
	}
	return false
}

// AccTestS3StorageUUID returns the UUID of an S3 storage destination for
// acceptance tests. Set COOLIFY_S3_STORAGE_UUID to the UUID of an S3
// storage registered in Coolify. The test is skipped if not set (or fails when
// COOLIFY_REQUIRE_TIP_APIS=1; tip bootstrap must create minio-test).
//
// Do not POST .../validate against this UUID: Coolify only allows the internal
// host coolify-minio for system storage uuid "minio". Validating bootstrap
// minio-test flips is_usable to false and breaks S3 backup acceptance tests.
func AccTestS3StorageUUID(t *testing.T) string {
	t.Helper()
	v := os.Getenv("COOLIFY_S3_STORAGE_UUID")
	if v == "" {
		accTestMissingFeature(t, "COOLIFY_S3_STORAGE_UUID not set (tip bootstrap should register minio-test S3 storage)")
	}
	return v
}

// AccTestSkipIfNoS3StorageAPI skips when Coolify lacks S3StoragesController
// (GET/POST /api/v1/s3-storages from Coolify >= v4.3.0).
//
// When COOLIFY_REQUIRE_TIP_APIS=1, a missing controller fails instead of skips.
//
// Probe strategy: GET /api/v1/s3-storages.
// Present controller returns 200 (list) or 401/403. Any 404 means no route
// (index never 404s when registered), including Coolify plain Not found.
func AccTestSkipIfNoS3StorageAPI(t *testing.T) {
	t.Helper()
	TestAccPreCheck(t)

	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	token := os.Getenv("COOLIFY_TOKEN")
	path := endpoint + "/api/v1/s3-storages"
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("building s3 storage probe request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		accTestMissingFeature(t, "s3 storage API probe failed (cannot reach Coolify): %v", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		return
	case http.StatusNotFound:
		// Index endpoint never 404s when the controller is registered; any 404
		// (including Coolify's plain {"message":"Not found."}) means no route.
		accTestMissingFeature(t, "Coolify instance has no S3StoragesController (HTTP %d). "+
			"Need Coolify >= v4.3.0. Body: %s",
			resp.StatusCode, truncateForSkip(string(body), 200))
		return
	case http.StatusMethodNotAllowed:
		accTestMissingFeature(t, "Coolify instance has no S3 storage GET route (HTTP 405). Need Coolify >= v4.3.0")
	default:
		if resp.StatusCode >= 500 {
			accTestMissingFeature(t, "s3 storage API probe returned HTTP %d (server error): %s",
				resp.StatusCode, truncateForSkip(string(body), 200))
		}
	}
}

// AccTestSkipIfNoNotificationAPI skips when Coolify lacks NotificationsController
// (GET /api/v1/notifications/discord from Coolify >= v4.3.0).
//
// When COOLIFY_REQUIRE_TIP_APIS=1, a missing controller fails instead of skips.
//
// Probe strategy: GET /api/v1/notifications/discord.
// Present controller returns 200 (settings) or 401/403. Any 404 means no route
// (including Coolify plain Not found).
func AccTestSkipIfNoNotificationAPI(t *testing.T) {
	t.Helper()
	TestAccPreCheck(t)

	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	token := os.Getenv("COOLIFY_TOKEN")
	path := endpoint + "/api/v1/notifications/discord"
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("building notification API probe request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		accTestMissingFeature(t, "notification API probe failed (cannot reach Coolify): %v", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		return
	case http.StatusNotFound:
		accTestMissingFeature(t, "Coolify instance has no NotificationsController (HTTP %d). "+
			"Need Coolify >= v4.3.0. Body: %s",
			resp.StatusCode, truncateForSkip(string(body), 200))
		return
	case http.StatusMethodNotAllowed:
		accTestMissingFeature(t, "Coolify instance has no notification GET route (HTTP 405). Need Coolify >= v4.3.0")
	default:
		if resp.StatusCode >= 500 {
			accTestMissingFeature(t, "notification API probe returned HTTP %d (server error): %s",
				resp.StatusCode, truncateForSkip(string(body), 200))
		}
	}
}

// AccTestSkipIfCoolifyBelow skips (or fails with COOLIFY_REQUIRE_TIP_APIS=1)
// when the connected Coolify version is older than min (e.g. "4.3.0").
func AccTestSkipIfCoolifyBelow(t *testing.T, min string) {
	t.Helper()
	TestAccPreCheck(t)
	c := AccTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ver, err := c.GetVersion(ctx)
	if err != nil {
		accTestMissingFeature(t, "could not read Coolify version: %v", err)
		return
	}
	if client.IsVersionAtLeast(ver, min) {
		return
	}
	accTestMissingFeature(t, "Coolify version %s is below required %s", ver, min)
}

// AccTestDockerfileAppConfig returns a Terraform config for an acceptance test
// of a Dockerfile application resource, including a project dependency. The
// extra parameter allows injecting additional HCL attributes.
func AccTestDockerfileAppConfig(name, serverUUID, extra string) string {
	return ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_application_dockerfile" "test" {
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
