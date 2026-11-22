package secretmanagersource_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSecretManagerSourceDataSource_Read(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{infToken: "infisical"}}
	m.linked = true
	m.token = infToken
	m.provider = "infisical"
	m.settings = map[string]string{
		"project_id":  "00000000-0000-4000-8000-000000000001",
		"environment": "prod",
		"secret_path": "/app",
	}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
data "coolify_secret_manager_source" "test" {
  application_uuid = %q
}
`, appUUID)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_secret_manager_source.test", "linked", "true"),
				resource.TestCheckResourceAttr("data.coolify_secret_manager_source.test", "integration_provider", "infisical"),
				resource.TestCheckResourceAttr("data.coolify_secret_manager_source.test", "infisical_secret_path", "/app"),
			),
		}},
	})
}

func TestSecretManagerSourceDataSource_Unlinked(t *testing.T) {
	t.Parallel()
	m := &sourceMock{}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
data "coolify_secret_manager_source" "test" {
  service_uuid = %q
}
`, svcUUID)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_secret_manager_source.test", "linked", "false"),
				resource.TestCheckNoResourceAttr("data.coolify_secret_manager_source.test", "integration_token_uuid"),
			),
		}},
	})
}

func TestSecretManagerSourceDataSource_MissingParent(t *testing.T) {
	t.Parallel()
	m := &sourceMock{get404: true}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
data "coolify_secret_manager_source" "test" {
  database_uuid = %q
}
`, dbUUID)),
			ExpectError: regexp.MustCompile("Error reading secret manager source"),
		}},
	})
}
