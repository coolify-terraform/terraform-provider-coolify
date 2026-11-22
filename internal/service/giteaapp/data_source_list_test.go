package giteaapp_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGiteaAppsDataSource(t *testing.T) {
	t.Parallel()
	body := `[{
		"id":7,
		"uuid":"550e8400-e29b-41d4-a716-446655440000",
		"name":"work",
		"organization":"acme",
		"api_url":"https://gitea.example/api/v1",
		"html_url":"https://gitea.example",
		"custom_user":"git",
		"custom_port":2222,
		"client_id":"client-1",
		"redirect_uri":"https://coolify.example/callback",
		"is_system_wide":false,
		"is_public":false,
		"is_connected":true,
		"team_id":3,
		"client_secret":"sekret",
		"access_token":"tok",
		"refresh_token":"ref",
		"webhook_secret":"wh",
		"created_at":"2026-11-01T00:00:00Z",
		"updated_at":"2026-11-02T00:00:00Z"
	},{
		"id":8,
		"uuid":"550e8400-e29b-41d4-a716-446655440001",
		"name":"other",
		"organization":null,
		"api_url":"https://other.example/api/v1",
		"html_url":"https://other.example",
		"custom_user":"",
		"custom_port":null,
		"client_id":null,
		"redirect_uri":null,
		"is_system_wide":true,
		"is_public":true,
		"is_connected":false,
		"team_id":null,
		"created_at":"2026-11-03T00:00:00Z",
		"updated_at":"2026-11-03T00:00:00Z"
	}]`
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gitea-apps" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}), "4.7.6"))
	t.Cleanup(srv.Close)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitea_apps" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.uuid", "550e8400-e29b-41d4-a716-446655440000"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.name", "work"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.is_connected", "true"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.custom_port", "2222"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.team_id", "3"),
					resource.TestCheckNoResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.client_secret"),
					resource.TestCheckNoResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.access_token"),
					resource.TestCheckNoResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.0.webhook_secret"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.1.is_connected", "false"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.test", "gitea_apps.1.is_system_wide", "true"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitea_apps" "filtered" {
  filter {
    name   = "name"
    values = ["work"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.filtered", "gitea_apps.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_gitea_apps.filtered", "gitea_apps.0.uuid", "550e8400-e29b-41d4-a716-446655440000"),
				),
			},
		},
	})
}

func TestGiteaAppsDataSource_OldCoolify(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	t.Cleanup(srv.Close)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitea_apps" "test" {}
`,
			ExpectError: regexp.MustCompile(`Coolify >= v4\.7\.0`),
		}},
	})
}

func TestGiteaAppsDataSource_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"uuid":"","name":"broken","is_connected":false,"is_system_wide":false,"is_public":false}]`))
	}), "4.7.0"))
	t.Cleanup(srv.Close)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_gitea_apps" "test" {}
`,
			ExpectError: regexp.MustCompile(`empty uuid`),
		}},
	})
}
