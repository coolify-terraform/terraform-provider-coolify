package environmentvariable_test

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_Create
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_Create(t *testing.T) {
	t.Parallel()
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
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
	t.Parallel()
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
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
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
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
	t.Parallel()
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: create so the resource exists in state.
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "IMPORT_VAR"
					value            = "import-value"
				`),
			},
			// Step 2: import and verify.
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateId:                        "application:cccc0001-0001-4000-8000-000000000001:env-import-uuid",
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
	t.Parallel()
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
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
// TestEnvironmentVariableResource_ServiceUpdate
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_ServiceUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentEnvVar := client.EnvironmentVariable{
		UUID: "env-svc-upd-uuid", Key: "LOG_LEVEL", Value: "info", IsPreview: false, IsBuild: false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentEnvVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{currentEnvVar})
	})
	mux.HandleFunc("PATCH /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["value"].(string); ok {
			currentEnvVar.Value = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					key          = "LOG_LEVEL"
					value        = "info"
				`),
				Check: resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "info"),
			},
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					key          = "LOG_LEVEL"
					value        = "debug"
				`),
				Check: resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "debug"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_ServiceImport
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_ServiceImport(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID: "env-svc-imp-uuid", Key: "SVC_VAR", Value: "svc-value", IsPreview: false, IsBuild: false,
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

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					key          = "SVC_VAR"
					value        = "svc-value"
				`),
			},
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateId:                        "service:ffff0001-0001-4000-8000-000000000001:env-svc-imp-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"value"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_ImportBadFormat
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_ImportBadFormat(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "env-err-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{{UUID: "env-err-uuid", Key: "K", Value: "V"}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "K"
					value            = "V"
				`),
			},
			{
				ResourceName:  "coolify_environment_variable.test",
				ImportState:   true,
				ImportStateId: "bad-format",
				ExpectError:   regexp.MustCompile(`Invalid import ID format`),
			},
		},
	})
}

func TestEnvironmentVariableResource_ImportBadType(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "env-err2-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{{UUID: "env-err2-uuid", Key: "K", Value: "V"}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "K"
					value            = "V"
				`),
			},
			{
				ResourceName:  "coolify_environment_variable.test",
				ImportState:   true,
				ImportStateId: "database:uuid:env-uuid",
				ExpectError:   regexp.MustCompile(`Invalid import ID type`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_Disappears
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_Disappears(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID:      "env-disappear-uuid",
		Key:       "DISAPPEAR_VAR",
		Value:     "some-value",
		IsPreview: false,
		IsBuild:   false,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "DISAPPEAR_VAR"
					value            = "some-value"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment_variable.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_environment_variable.test", "/api/v1/applications/cccc0001-0001-4000-8000-000000000001/envs/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_ServiceDisappears
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_ServiceDisappears(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID: "env-svc-disappear-uuid", Key: "SVC_GONE", Value: "val", IsPreview: false, IsBuild: false,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					key          = "SVC_GONE"
					value        = "val"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment_variable.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_environment_variable.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/services/ffff0001-0001-4000-8000-000000000001/envs/"+uuid, nil)
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_InvalidKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "1INVALID"
					value            = "test"
				`),
				ExpectError: regexp.MustCompile(`valid environment variable name`),
			},
		},
	})
}

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
