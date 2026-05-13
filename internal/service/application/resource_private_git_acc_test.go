package application_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateGitApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-privgit")
	privKey := generateTestRSAKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_private_git_application", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccPrivateGitAppConfig(name, serverUUID, privKey, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_git_application.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "git_repository", "git@github.com:coollabsio/coolify-examples.git"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_private_git_application.test", "ports_exposes", "3000"),
				),
			},
			// Step 2: Update description
			{
				Config: testAccPrivateGitAppConfig(name, serverUUID, privKey, `description = "Updated private git app"`),
				Check:  resource.TestCheckResourceAttr("coolify_private_git_application.test", "description", "Updated private git app"),
			},
			// Step 3: Import by UUID
			{
				ResourceName:                         "coolify_private_git_application.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_private_git_application.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name", "private_key_uuid", "project_uuid", "server_uuid"},
			},
		},
	})
}

func generateTestRSAKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %s", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}

func testAccPrivateGitAppConfig(name, serverUUID, privKey, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_private_key" "test" {
  name        = "%[1]s-key"
  private_key = %[3]q
}

resource "coolify_private_git_application" "test" {
  project_uuid     = coolify_project.test.uuid
  server_uuid      = %[2]q
  private_key_uuid = coolify_private_key.test.uuid
  git_repository   = "git@github.com:coollabsio/coolify-examples.git"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  %[4]s
}
`, name, serverUUID, privKey, extra)
}
