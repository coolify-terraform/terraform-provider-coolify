package environmentvariable_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
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

func TestEnvironmentVariablesDataSource_Database(t *testing.T) {
	t.Parallel()
	envVars := []client.EnvironmentVariable{
		{UUID: "ev-d1", Key: "POSTGRES_USER", Value: "admin", IsPreview: false, IsBuild: false},
		{UUID: "ev-d2", Key: "POSTGRES_PASSWORD", Value: "secret", IsPreview: false, IsBuild: false},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
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
  database_uuid = "dddd0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.key", "POSTGRES_USER"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.1.key", "POSTGRES_PASSWORD"),
				),
			},
		},
	})
}

func TestEnvironmentVariablesDataSource_PreservesRawPreviewAndNonPreviewRows(t *testing.T) {
	t.Parallel()
	envVars := []client.EnvironmentVariable{
		{UUID: "ev-preview", Key: "DB_HOST", Value: "preview-host", IsPreview: true, IsBuild: false},
		{UUID: "ev-runtime", Key: "DB_HOST", Value: "runtime-host", IsPreview: false, IsBuild: false},
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
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.uuid", "ev-preview"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.key", "DB_HOST"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.value", "preview-host"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.0.is_preview", "true"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.1.uuid", "ev-runtime"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.1.value", "runtime-host"),
					resource.TestCheckResourceAttr("data.coolify_environment_variables.test", "environment_variables.1.is_preview", "false"),
				),
			},
		},
	})
}

func TestEnvironmentVariablesDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: `data "coolify_environment_variables" "test" {
  application_uuid = "not-a-valid-uuid"
}
`,
			ExpectError: acctest.UUIDValidationError(),
		}},
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
