package privatekey_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateKeyResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-pk")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_private_key", "/api/v1/security/keys/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = %[1]q
  description = "acc test key"
  private_key = %[2]q
}
`, name, privKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", name),
					resource.TestCheckResourceAttr("coolify_private_key.test", "description", "acc test key"),
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "public_key"),
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "fingerprint"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "is_git_related", "false"),
				),
			},
			// Idempotency check
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = %[1]q
  description = "acc test key"
  private_key = %[2]q
}
`, name, privKey),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 2: Update name and description
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = %[1]q
  description = "updated acc test key"
  private_key = %[2]q
}
`, name+"-updated", privKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "description", "updated acc test key"),
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "public_key"),
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "fingerprint"),
				),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_private_key.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_private_key.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"private_key"},
			},
		},
	})
}

func TestAccPrivateKeyDataSources(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	name := acctest.RandomWithPrefix("tf-acc-pk-ds")
	privKey := acctest.GenerateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_private_key", "/api/v1/security/keys/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_private_key" "test" {
  name        = %[1]q
  description = "acc test key for data sources"
  private_key = %[2]q
}

data "coolify_private_key" "by_uuid" {
  uuid = coolify_private_key.test.uuid
}

data "coolify_private_keys" "all" {
  depends_on = [coolify_private_key.test]
}
`, name, privKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify singular data source
					resource.TestCheckResourceAttrPair("data.coolify_private_key.by_uuid", "uuid", "coolify_private_key.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.coolify_private_key.by_uuid", "name", "coolify_private_key.test", "name"),
					resource.TestCheckResourceAttrPair("data.coolify_private_key.by_uuid", "public_key", "coolify_private_key.test", "public_key"),
					resource.TestCheckResourceAttrPair("data.coolify_private_key.by_uuid", "fingerprint", "coolify_private_key.test", "fingerprint"),
					// Verify list data source
					resource.TestMatchResourceAttr("data.coolify_private_keys.all", "private_keys.#", regexp.MustCompile(`[1-9]\d*`)),
				),
			},
		},
	})
}
