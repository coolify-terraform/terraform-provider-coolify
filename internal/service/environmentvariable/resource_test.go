package environmentvariable_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_Create
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_Create(t *testing.T) {
	envVar := client.EnvironmentVariable{
		UUID:      "env-create-uuid",
		Key:       "DATABASE_URL",
		Value:     "postgres://localhost/mydb",
		IsPreview: false,
		IsBuild:   false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "app-uuid-1"
					key              = "DATABASE_URL"
					value            = "postgres://localhost/mydb"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "uuid", "env-create-uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "DATABASE_URL"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "postgres://localhost/mydb"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_preview", "false"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "false"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_Update
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_Update(t *testing.T) {
	mu := sync.Mutex{}
	currentEnvVar := client.EnvironmentVariable{
		UUID:      "env-update-uuid",
		Key:       "API_KEY",
		Value:     "initial-secret",
		IsPreview: false,
		IsBuild:   false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentEnvVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{currentEnvVar})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["value"].(string); ok {
			currentEnvVar.Value = v
		}
		if v, ok := body["is_build_time"].(bool); ok {
			currentEnvVar.IsBuild = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "app-uuid-1"
					key              = "API_KEY"
					value            = "initial-secret"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "initial-secret"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "false"),
				),
			},
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "app-uuid-1"
					key              = "API_KEY"
					value            = "updated-secret"
					is_build         = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "updated-secret"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "true"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_Import
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_Import(t *testing.T) {
	envVar := client.EnvironmentVariable{
		UUID:      "env-import-uuid",
		Key:       "IMPORT_VAR",
		Value:     "import-value",
		IsPreview: false,
		IsBuild:   false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: create so the resource exists in state.
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "app-uuid-1"
					key              = "IMPORT_VAR"
					value            = "import-value"
				`),
			},
			// Step 2: import and verify.
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateId:                        "application:app-uuid-1:env-import-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"value"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_CreateWithServiceUUID
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_CreateWithServiceUUID(t *testing.T) {
	envVar := client.EnvironmentVariable{
		UUID:      "env-svc-uuid",
		Key:       "REDIS_URL",
		Value:     "redis://localhost:6379",
		IsPreview: true,
		IsBuild:   true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "svc-uuid-1"
					key          = "REDIS_URL"
					value        = "redis://localhost:6379"
					is_preview   = true
					is_build     = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "uuid", "env-svc-uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "REDIS_URL"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "redis://localhost:6379"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_preview", "true"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "true"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testEnvVarResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_environment_variable" "test" {
  %s
}
`, endpoint, attrs)
}
