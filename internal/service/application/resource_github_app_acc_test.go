package application_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGitHubAppApplicationResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	fixture := accTestGitHubAppApplicationFixture(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-ghapp-app")
	privateKeyName := acctest.RandomWithPrefix("tf-acc-ghapp-app-key")
	// Generate a unique RSA key to avoid fingerprint collisions with the
	// scenario tests that register the real GitHub App PEM concurrently.
	privKey := acctest.GenerateTestRSAKey(t)
	updatedDescription := "Updated github app application"
	createConfig := testAccGitHubAppApplicationConfig(name, serverUUID, privateKeyName, privKey, fixture, "")
	updatedConfig := testAccGitHubAppApplicationConfig(
		name,
		serverUUID,
		privateKeyName,
		privKey,
		fixture,
		fmt.Sprintf(`description = %q`, updatedDescription),
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_application_github_app", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: createConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_github_app.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("coolify_application_github_app.test", "ports_exposes", "3000"),
				),
			},
			// Step 2: Idempotency check after create
			{
				Config:             createConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 3: Update description
			{
				Config: updatedConfig,
				Check:  resource.TestCheckResourceAttr("coolify_application_github_app.test", "description", updatedDescription),
			},
			// Step 4: Idempotency check
			{
				Config:             updatedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 5: Import by UUID
			{
				ResourceName:                         "coolify_application_github_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_application_github_app.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"environment_name", "github_app_uuid", "project_uuid", "server_uuid"}, // github_app_uuid is not returned by the API after import
			},
		},
	})
}

type gitHubAppApplicationFixture struct {
	AppID          int64
	InstallationID int64
	ClientID       string
	ClientSecret   string
	GitRepository  string
	GitBranch      string
}

func accTestGitHubAppApplicationFixture(t *testing.T) gitHubAppApplicationFixture {
	t.Helper()

	required := []string{
		"COOLIFY_GITHUB_APP_APP_ID",
		"COOLIFY_GITHUB_APP_INSTALLATION_ID",
		"COOLIFY_GITHUB_APP_CLIENT_ID",
		"COOLIFY_GITHUB_APP_CLIENT_SECRET",
		"COOLIFY_GITHUB_APP_REPOSITORY",
	}
	missing := make([]string, 0, len(required))
	values := make(map[string]string, len(required))
	for _, key := range required {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			missing = append(missing, key)
			continue
		}
		values[key] = value
	}

	if len(missing) > 0 {
		t.Skipf("GitHub App application acceptance requires live GitHub App credentials and repository access; missing %s", strings.Join(missing, ", "))
	}

	appID, err := strconv.ParseInt(values["COOLIFY_GITHUB_APP_APP_ID"], 10, 64)
	if err != nil {
		t.Fatalf("parsing COOLIFY_GITHUB_APP_APP_ID: %s", err)
	}
	installationID, err := strconv.ParseInt(values["COOLIFY_GITHUB_APP_INSTALLATION_ID"], 10, 64)
	if err != nil {
		t.Fatalf("parsing COOLIFY_GITHUB_APP_INSTALLATION_ID: %s", err)
	}

	branch := strings.TrimSpace(os.Getenv("COOLIFY_GITHUB_APP_BRANCH"))
	if branch == "" {
		branch = "main"
	}

	return gitHubAppApplicationFixture{
		AppID:          appID,
		InstallationID: installationID,
		ClientID:       values["COOLIFY_GITHUB_APP_CLIENT_ID"],
		ClientSecret:   values["COOLIFY_GITHUB_APP_CLIENT_SECRET"],
		GitRepository:  values["COOLIFY_GITHUB_APP_REPOSITORY"],
		GitBranch:      branch,
	}
}

func testAccGitHubAppApplicationConfig(name, serverUUID, privateKeyName, privKey string, fixture gitHubAppApplicationFixture, extra string) string {
	return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}

resource "coolify_private_key" "test" {
  name        = %[3]q
  description = "acc test github app application key"
  private_key = %[4]q
}

resource "coolify_github_app" "test" {
  name             = "%[1]s-ghapp"
  app_id           = %[5]d
  installation_id  = %[6]d
  client_id        = %[7]q
  client_secret    = %[8]q
  private_key_uuid = coolify_private_key.test.uuid
}

resource "coolify_application_github_app" "test" {
  project_uuid    = coolify_project.test.uuid
  server_uuid     = %[2]q
  github_app_uuid = coolify_github_app.test.uuid
  git_repository  = %[9]q
  git_branch      = %[10]q
  build_pack      = "nixpacks"
  ports_exposes   = "3000"
  %[11]s
}
`, name, serverUUID, privateKeyName, privKey, fixture.AppID, fixture.InstallationID, fixture.ClientID, fixture.ClientSecret, fixture.GitRepository, fixture.GitBranch, extra)
}
