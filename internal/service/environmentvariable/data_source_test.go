package environmentvariable_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentVariablesDataSource_Application(t *testing.T) {
	t.Parallel()
	envVars := []client.EnvironmentVariable{
		{UUID: "ev-1", Key: "DB_HOST", Value: "localhost", IsPreview: false, IsBuild: false},
		{UUID: "ev-2", Key: "DB_PORT", Value: "5432", IsPreview: false, IsBuild: true},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envVars)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_environment_variables" "test" {
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.key", "DB_HOST"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.value", "localhost"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.1.key", "DB_PORT"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.1.is_build", "true"),
				),
			},
		},
	})
}

func TestEnvironmentVariablesDataSource_Service(t *testing.T) {
	t.Parallel()
	envVars := []client.EnvironmentVariable{
		{UUID: "ev-s1", Key: "REDIS_URL", Value: "redis://localhost", IsPreview: true, IsBuild: false},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envVars)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_environment_variables" "test" {
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.key", "REDIS_URL"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.is_preview", "true"),
				),
			},
		},
	})
}
