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
		{UUID: "ev-1", Key: "DB_HOST", Value: "localhost", IsPreview: false, IsBuild: false},
		{UUID: "ev-2", Key: "DB_PORT", Value: "5432", IsPreview: true, IsBuild: true},
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
  uuid             = "ev-2"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "uuid", "ev-2"),
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
		{UUID: "ev-s1", Key: "REDIS_URL", Value: "redis://localhost", IsPreview: false, IsBuild: false},
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
  uuid         = "ev-s1"
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "uuid", "ev-s1"),
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
		{UUID: "ev-d1", Key: "POSTGRES_USER", Value: "admin", IsPreview: false, IsBuild: false},
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
  uuid          = "ev-d1"
  database_uuid = "dddd0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "uuid", "ev-d1"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "key", "POSTGRES_USER"),
					resource.TestCheckResourceAttr("data.coolify_environment_variable.test", "value", "admin"),
				),
			},
		},
	})
}

func TestEnvironmentVariableDataSource_NotFound(t *testing.T) {
	t.Parallel()

	envVars := []client.EnvironmentVariable{
		{UUID: "ev-1", Key: "DB_HOST", Value: "localhost", IsPreview: false, IsBuild: false},
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
  uuid             = "nonexistent-uuid"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`not found`),
			},
		},
	})
}
