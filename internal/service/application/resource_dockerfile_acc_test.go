package application_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestAccDockerfileApplicationResource_CRUD
// ---------------------------------------------------------------------------

func TestAccDockerfileApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkrfile")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_dockerfile", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.AccTestDockerfileAppConfig(name, serverUUID, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_dockerfile.test", "uuid"),
					resource.TestCheckResourceAttrSet("coolify_application_dockerfile.test", "dockerfile_location"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "ports_exposes", "80"),
				),
			},
			// Idempotency check
			{
				Config:             acctest.AccTestDockerfileAppConfig(name, serverUUID, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update description
			{
				Config: acctest.AccTestDockerfileAppConfig(name, serverUUID, `description = "Updated via acc test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "description", "Updated via acc test"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_application_dockerfile.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_dockerfile.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name", "project_uuid", "server_uuid", "dockerfile_location"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccDockerfileApplicationDataSources
// ---------------------------------------------------------------------------

func TestAccDockerfileApplicationDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkrfile-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_dockerfile", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testAccDockerfileAppWithDataSourcesConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the resource was created
					resource.TestCheckResourceAttrSet("coolify_application_dockerfile.test", "uuid"),
					// Verify the singular data source reads the application
					resource.TestCheckResourceAttrPair(
						"data.coolify_application.test", "uuid",
						"coolify_application_dockerfile.test", "uuid",
					),
					resource.TestCheckResourceAttrPair(
						"data.coolify_application.test", "name",
						"coolify_application_dockerfile.test", "name",
					),
					// Verify the list data source contains at least one application
					resource.TestCheckResourceAttrSet("data.coolify_applications.test", "applications.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccDockerfileAppWithDataSourcesConfig(name, serverUUID string) string {
	return acctest.AccTestDockerfileAppConfig(name, serverUUID, "") + `
data "coolify_application" "test" {
  uuid = coolify_application_dockerfile.test.uuid
}

data "coolify_applications" "test" {
  depends_on = [coolify_application_dockerfile.test]
}
`
}

// ---------------------------------------------------------------------------
// TestAccApplicationLogsDataSource
// ---------------------------------------------------------------------------

func TestAccApplicationLogsDataSource(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-applogs")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_dockerfile", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testAccAppLogsConfig(name, serverUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.#", "0"),
				),
			},
		},
	})
}

func testAccAppLogsConfig(name, serverUUID string) string {
	return acctest.AccTestDockerfileAppConfig(name, serverUUID, "") + `
data "coolify_application_logs" "test" {
  uuid = coolify_application_dockerfile.test.uuid
}
`
}

// TestAccDockerfileApplicationResource_V43Settings exercises Coolify >= v4.3.0
// application write fields (is_log_drain_enabled, noindex_domains) against a
// real instance. On tip-edge CI (COOLIFY_REQUIRE_TIP_APIS=1) the version probe
// fails the run if Coolify is older than 4.3.0; elsewhere it skips.
func TestAccDockerfileApplicationResource_V43Settings(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	acctest.AccTestSkipIfCoolifyBelow(t, "4.3.0")
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-dkr-v43")

	// Domain must match noindex entry (Coolify ignores noindex URLs not in domains).
	extra := `
  domains              = "http://acc-v43.example.com"
  is_log_drain_enabled = true
  noindex_domains      = ["http://acc-v43.example.com"]
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_dockerfile", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDockerfileAppConfig(name, serverUUID, extra),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_dockerfile.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "is_log_drain_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "noindex_domains.#", "1"),
					resource.TestCheckResourceAttr("coolify_application_dockerfile.test", "noindex_domains.0", "http://acc-v43.example.com"),
				),
			},
			{
				Config:             acctest.AccTestDockerfileAppConfig(name, serverUUID, extra),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
