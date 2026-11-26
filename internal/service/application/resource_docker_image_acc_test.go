package application_test

import (
	"fmt"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestAccDockerImageApplicationResource_CRUD
// ---------------------------------------------------------------------------

func TestAccDockerImageApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkrimg")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, "nginx:alpine", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_docker_image.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "name", name),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:alpine"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "80"),
				),
			},
			// Idempotency check
			{
				Config:             testAccDockerImageAppConfig(name, serverUUID, "nginx:alpine", ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update docker_image (create accepts image:tag; PATCH name is name-only)
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, "nginx:1.27-alpine", `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:1.27-alpine"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_application_docker_image.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_docker_image.test", "uuid"),
				// docker_image is ignored because Coolify may strip the tag (e.g. "nginx:alpine" → "nginx")
				// and the provider cannot reconstruct the original tag on import.
				ImportStateVerifyIgnore: []string{"environment_name", "project_uuid", "server_uuid", "docker_image", "container_present"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccDockerImageAppConfig(name, serverUUID, image, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_application_docker_image" "test" {
  project_uuid  = coolify_project.test.uuid
  server_uuid   = %[2]q
  name          = %[1]q
  docker_image  = %[3]q
  ports_exposes = "80"
  %[4]s
}
`, name, serverUUID, image, extra)
}

// TestAccDockerImageApplicationResource_NoPublicDomain creates an internal app
// with autogenerate_domain=false and no domains; Coolify must not assign sslip FQDN.
func TestAccDockerImageApplicationResource_NoPublicDomain(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-nogen")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_docker_image" "test" {
  project_uuid         = coolify_project.test.uuid
  server_uuid          = %q
  name                 = %q
  docker_image         = "nginx:alpine"
  ports_exposes        = "80"
  autogenerate_domain  = false
}
`, name, serverUUID, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_docker_image.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "autogenerate_domain", "false"),
					// Empty or unset FQDN: Coolify may return "" or omit; accept empty string.
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains", ""),
				),
			},
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_docker_image" "test" {
  project_uuid         = coolify_project.test.uuid
  server_uuid          = %q
  name                 = %q
  docker_image         = "nginx:alpine"
  ports_exposes        = "80"
  autogenerate_domain  = false
}
`, name, serverUUID, name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccDockerImageApplicationResource_ClearDomains updates domains="" and
// expects Coolify to clear fqdn. The live probe skips when the instance
// ignores empty PATCH (provider #647, coollabsio/coolify#11116). Provider
// wire format is covered by TestDockerImageApplicationResource_ClearDomainsOnUpdate.
func TestAccDockerImageApplicationResource_ClearDomains(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCannotClearApplicationDomains(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-clrdmn")
	domain := fmt.Sprintf("http://%s.example.com", name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, "nginx:alpine", fmt.Sprintf(`
  autogenerate_domain = false
  domains             = %q
`, domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains", domain),
				),
			},
			{
				Config: testAccDockerImageAppConfig(name, serverUUID, "nginx:alpine", `
  autogenerate_domain = false
  domains             = ""
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains", ""),
				),
			},
		},
	})
}

// TestAccDockerImageApplicationResource_ZeroPorts creates a docker image app
// with no ports_exposes (zero-port app), then sets and clears the ports.
// Covers the wire forms end to end: create omits the key, clearing sends an
// explicit null (Coolify rejects "").
func TestAccDockerImageApplicationResource_ZeroPorts(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	// The clear step sends PATCH null, which Coolify only accepts from v4.3.15.
	// The nightly floor image is v4.1.2, so skip there rather than 422.
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.15")
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-zeroport")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create a zero-port app (ports_exposes omitted).
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_docker_image" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %q
  name                = %q
  docker_image        = "nginx:alpine"
  autogenerate_domain = false
}
`, name, serverUUID, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_docker_image.test", "uuid"),
				),
			},
			// Step 2: Expose a port.
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_docker_image" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %q
  name                = %q
  docker_image        = "nginx:alpine"
  ports_exposes       = "9252"
  autogenerate_domain = false
}
`, name, serverUUID, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "9252"),
				),
			},
			// Step 3: Clear the ports back to a zero-port app.
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_docker_image" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %q
  name                = %q
  docker_image        = "nginx:alpine"
  autogenerate_domain = false
}
`, name, serverUUID, name),
			},
			// Idempotency check
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %q
}

resource "coolify_application_docker_image" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %q
  name                = %q
  docker_image        = "nginx:alpine"
  autogenerate_domain = false
}
`, name, serverUUID, name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
