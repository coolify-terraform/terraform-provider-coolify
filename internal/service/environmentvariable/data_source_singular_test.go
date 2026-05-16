package environmentvariable_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentVariableDataSource_Application(t *testing.T) {
	t.Parallel()

	envVars := []client.EnvironmentVariable{
		{UUID: "11111111-1111-4111-8111-111111111111", Key: "DB_HOST", Value: "localhost", IsPreview: false, IsBuild: false},
		{UUID: "22222222-2222-4222-8222-222222222222", Key: "DB_PORT", Value: "5432", IsPreview: true, IsBuild: true},
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
data "coolify_environment_variable" "test" {
  uuid             = "22222222-2222-4222-8222-222222222222"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "uuid", "22222222-2222-4222-8222-222222222222"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "key", "DB_PORT"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "value", "5432"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "is_preview", "true"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "is_build", "true"),
				),
			},
		},
	})
}

func TestEnvironmentVariableDataSource_Service(t *testing.T) {
	t.Parallel()

	envVars := []client.EnvironmentVariable{
		{UUID: "33333333-3333-4333-8333-333333333333", Key: "REDIS_URL", Value: "redis://localhost", IsPreview: false, IsBuild: false},
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
data "coolify_environment_variable" "test" {
  uuid         = "33333333-3333-4333-8333-333333333333"
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "uuid", "33333333-3333-4333-8333-333333333333"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "key", "REDIS_URL"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "value", "redis://localhost"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "is_preview", "false"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "is_build", "false"),
				),
			},
		},
	})
}

func TestEnvironmentVariableDataSource_Database(t *testing.T) {
	t.Parallel()

	envVars := []client.EnvironmentVariable{
		{UUID: "44444444-4444-4444-8444-444444444444", Key: "POSTGRES_USER", Value: "admin", IsPreview: false, IsBuild: false},
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
data "coolify_environment_variable" "test" {
  uuid          = "44444444-4444-4444-8444-444444444444"
  database_uuid = "dddd0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "uuid", "44444444-4444-4444-8444-444444444444"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "key", "POSTGRES_USER"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "value", "admin"),
				),
			},
		},
	})
}

func TestEnvironmentVariableDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: `data "coolify_environment_variable" "test" {
  uuid             = "not-a-valid-uuid"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: acctest.UUIDValidationError(),
		}},
	})
}

func TestEnvironmentVariableDataSource_NotFound(t *testing.T) {
	t.Parallel()

	envVars := []client.EnvironmentVariable{
		{UUID: "11111111-1111-4111-8111-111111111111", Key: "DB_HOST", Value: "localhost", IsPreview: false, IsBuild: false},
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
data "coolify_environment_variable" "test" {
  uuid             = "55555555-5555-4555-8555-555555555555"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`not\s+found`),
			},
		},
	})
}
