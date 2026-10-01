package serverregistry_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerDockerRegistry_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	skipIfNoRegistryAPI(t, serverUUID)
	registry, username, password := registryCredentials(t)

	config := func(user string) string {
		return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_server_docker_registry" "test" {
  server_uuid = %q
  registry    = %q
  username    = %q
  password    = %q
}
`, serverUUID, registry, user, password)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(username),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "registry", registry),
					resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "username", username),
				),
			},
			{
				Config: config(username),
				Check:  resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "username", username),
			},
			{
				ResourceName:            "coolify_server_docker_registry.test",
				ImportState:             true,
				ImportStateId:           serverUUID + ":" + registry,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// registryCredentials skips when no real registry login is configured.
// Coolify runs docker login on the server. A placeholder password returns
// HTTP 400 once the route exists (CI edge).
func registryCredentials(t *testing.T) (registry, username, password string) {
	t.Helper()
	username = os.Getenv("COOLIFY_DOCKER_REGISTRY_USERNAME")
	password = os.Getenv("COOLIFY_DOCKER_REGISTRY_PASSWORD")
	registry = os.Getenv("COOLIFY_DOCKER_REGISTRY")
	if registry == "" {
		registry = "ghcr.io"
	}
	if username == "" || password == "" {
		t.Skip("COOLIFY_DOCKER_REGISTRY_USERNAME and COOLIFY_DOCKER_REGISTRY_PASSWORD not set; Coolify runs docker login on the server")
	}
	return registry, username, password
}

func TestAccServerDockerRegistriesDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	skipIfNoRegistryAPI(t, serverUUID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
data "coolify_server_docker_registries" "test" {
  server_uuid = %q
}
`, serverUUID),
			Check: resource.TestCheckResourceAttrSet("data.coolify_server_docker_registries.test", "registries.#"),
		}},
	})
}

func skipIfNoRegistryAPI(t *testing.T, serverUUID string) {
	t.Helper()
	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	req, err := http.NewRequest(http.MethodPost, endpoint+"/api/v1/servers/"+serverUUID+"/registries", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("COOLIFY_TOKEN"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("registry probe failed: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		t.Skipf("Coolify has no POST /servers/{uuid}/registries (HTTP %d): %s", resp.StatusCode, raw)
	}
}
