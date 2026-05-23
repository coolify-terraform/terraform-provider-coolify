package cloudtoken_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestAccCloudTokenResource_CRUD
// ---------------------------------------------------------------------------

func TestAccCloudTokenResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	token := os.Getenv("COOLIFY_HETZNER_TOKEN")
	if token == "" {
		t.Skip("COOLIFY_HETZNER_TOKEN not set, skipping (Coolify validates token against Hetzner API)")
	}
	name := acctest.RandomWithPrefix("tf-acc-ctoken")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_cloud_token", "/api/v1/cloud-tokens/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccCloudTokenConfig(name, "hetzner", token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_cloud_token.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "name", name),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "cloud_provider", "hetzner"),
				),
			},
			// Idempotency check
			{
				Config:             testAccCloudTokenConfig(name, "hetzner", token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update name
			{
				Config: testAccCloudTokenConfig(name+"-updated", "hetzner", token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "cloud_provider", "hetzner"),
				),
			},
			// Step 3: Import
			{
				ResourceName:                         "coolify_cloud_token.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_cloud_token.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"token"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestAccCloudTokenDataSources
// ---------------------------------------------------------------------------

func TestAccCloudTokenDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	if os.Getenv("COOLIFY_HETZNER_TOKEN") == "" {
		t.Skip("COOLIFY_HETZNER_TOKEN not set, skipping (Coolify validates token against Hetzner API)")
	}
	name := acctest.RandomWithPrefix("tf-acc-ctoken-ds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_cloud_token", "/api/v1/cloud-tokens/"),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudTokenWithDataSourcesConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the resource was created
					resource.TestCheckResourceAttrSet("coolify_cloud_token.test", "uuid"),
					// Verify the singular data source reads the cloud token
					resource.TestCheckResourceAttrPair(
						"data.coolify_cloud_token.test", "uuid",
						"coolify_cloud_token.test", "uuid",
					),
					resource.TestCheckResourceAttrPair(
						"data.coolify_cloud_token.test", "name",
						"coolify_cloud_token.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.coolify_cloud_token.test", "cloud_provider",
						"coolify_cloud_token.test", "cloud_provider",
					),
					// Verify the list data source contains at least one cloud token
					resource.TestCheckResourceAttrSet("data.coolify_cloud_tokens.test", "cloud_tokens.#"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testAccCloudTokenConfig(name, cloudProvider, token string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_token" "test" {
  name           = %[1]q
  cloud_provider = %[2]q
  token          = %[3]q
}
`, name, cloudProvider, token)
}

func testAccCloudTokenWithDataSourcesConfig(name string) string {
	token := os.Getenv("COOLIFY_HETZNER_TOKEN")
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_cloud_token" "test" {
  name           = %[1]q
  cloud_provider = "hetzner"
  token          = %[2]q
}

data "coolify_cloud_token" "test" {
  uuid = coolify_cloud_token.test.uuid
}

data "coolify_cloud_tokens" "test" {
  depends_on = [coolify_cloud_token.test]
}
`, name, token)
}
