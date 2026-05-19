package environmentvariable_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		IsBuild:   true,
	}

	mu := sync.Mutex{}
	deleted := false
	createBuildtimePresent := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		_, createBuildtimePresent = body["is_buildtime"]
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkEnvVarDestroy(srv.URL, "applications", "cccc0001-0001-4000-8000-000000000001"),
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
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "true"),
					resource.TestCheckFunc(func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if createBuildtimePresent {
							return fmt.Errorf("expected create request to omit is_buildtime when is_build is omitted")
						}
						return nil
					}),
				),
			},
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "DATABASE_URL"
					value            = "postgres://localhost/mydb"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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
		IsBuild:   true,
	}
	updateBuildtimePresent := false
	updateBuildtimeValue := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentEnvVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{currentEnvVar})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["value"].(string); ok {
			currentEnvVar.Value = v
		}
		if v, ok := body["is_buildtime"].(bool); ok {
			updateBuildtimePresent = true
			updateBuildtimeValue = v
			currentEnvVar.IsBuild = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != currentEnvVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "true"),
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
					resource.TestCheckFunc(func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if !updateBuildtimePresent {
							return fmt.Errorf("expected update request to include is_buildtime")
						}
						if !updateBuildtimeValue {
							return fmt.Errorf("expected update request to send is_buildtime=true")
						}
						return nil
					}),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_Import
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_ReadPreservesValueWhenAPIHidesIt(t *testing.T) {
	t.Parallel()
	currentEnvVar := client.EnvironmentVariable{
		UUID:      "env-hidden-uuid",
		Key:       "SECRET_KEY",
		Value:     "initial-secret",
		IsPreview: false,
		IsBuild:   true,
	}
	returnHiddenValue := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentEnvVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := currentEnvVar
		if returnHiddenValue {
			resp.Value = ""
		}
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{resp})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["value"].(string); ok {
			currentEnvVar.Value = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != currentEnvVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
					key              = "SECRET_KEY"
					value            = "initial-secret"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "initial-secret"),
					resource.TestCheckFunc(func(s *terraform.State) error {
						returnHiddenValue = true
						return nil
					}),
				),
			},
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					key              = "SECRET_KEY"
					value            = "initial-secret"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestEnvironmentVariableResource_Import(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID:      "eeee0001-0001-4000-8000-000000000001",
		Key:       "IMPORT_VAR",
		Value:     "import-value",
		IsPreview: false,
		IsBuild:   true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
				ImportStateId:                        "application:cccc0001-0001-4000-8000-000000000001:eeee0001-0001-4000-8000-000000000001",
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
		IsBuild:   false,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkEnvVarDestroy(srv.URL, "services", "ffff0001-0001-4000-8000-000000000001"),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					key          = "REDIS_URL"
					value        = "redis://localhost:6379"
					is_preview   = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "uuid", "env-svc-uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "REDIS_URL"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "redis://localhost:6379"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_preview", "true"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "false"),
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
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentEnvVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{currentEnvVar})
	})
	mux.HandleFunc("PATCH /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["value"].(string); ok {
			currentEnvVar.Value = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != currentEnvVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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

func TestEnvironmentVariableResource_ServiceBuildVarRejected(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig("https://example.invalid", `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					key          = "LOG_LEVEL"
					value        = "debug"
					is_build     = true
				`),
				ExpectError: regexp.MustCompile("application-scoped environment variables"),
			},
		},
	})
}

func TestEnvironmentVariableResource_ServiceImport(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID: "eeee0002-0002-4000-8000-000000000001", Key: "SVC_VAR", Value: "svc-value", IsPreview: false, IsBuild: false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
				ImportStateId:                        "service:ffff0001-0001-4000-8000-000000000001:eeee0002-0002-4000-8000-000000000001",
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
		json.NewEncoder(w).Encode(map[string]string{"uuid": "eeee0002-0002-4000-8000-000000000001"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{{UUID: "eeee0002-0002-4000-8000-000000000001", Key: "K", Value: "V"}})
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
				ImportStateId: "unknown:aaaa0001-0001-4000-8000-000000000001:bbbb0001-0001-4000-8000-000000000001",
				ExpectError:   regexp.MustCompile(`Invalid import ID type`),
			},
		},
	})
}

func TestEnvironmentVariableResource_ImportBadParentUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "eeee0003-0003-4000-8000-000000000001"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{{UUID: "eeee0003-0003-4000-8000-000000000001", Key: "K", Value: "V"}})
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
				ImportStateId: "application:not-a-uuid:eeee0003-0003-4000-8000-000000000001",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*parent UUID segment`),
			},
		},
	})
}

func TestEnvironmentVariableResource_ImportBadEnvUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "eeee0004-0004-4000-8000-000000000001"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{{UUID: "eeee0004-0004-4000-8000-000000000001", Key: "K", Value: "V"}})
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
				ImportStateId: "application:aaaa0001-0001-4000-8000-000000000001:not-a-uuid",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*env variable UUID segment`),
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
		IsBuild:   true,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_DatabaseDisappears
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_DatabaseDisappears(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID: "env-db-disappear-uuid", Key: "DB_GONE", Value: "val", IsPreview: false, IsBuild: false,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/databases/{dbUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
					database_uuid = "dddd0001-0001-4000-8000-000000000001"
					key           = "DB_GONE"
					value         = "val"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_environment_variable.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_environment_variable.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/databases/dddd0001-0001-4000-8000-000000000001/envs/"+uuid, nil)
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
// TestEnvironmentVariableResource_CreateDatabase
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_CreateDatabase(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID:      "env-db-uuid",
		Key:       "POSTGRES_PASSWORD",
		Value:     "supersecret",
		IsPreview: false,
		IsBuild:   false,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{})
		} else {
			json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
		}
	})
	mux.HandleFunc("DELETE /api/v1/databases/{dbUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkEnvVarDestroy(srv.URL, "databases", "dddd0001-0001-4000-8000-000000000001"),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					database_uuid = "dddd0001-0001-4000-8000-000000000001"
					key           = "POSTGRES_PASSWORD"
					value         = "supersecret"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "uuid", "env-db-uuid"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "key", "POSTGRES_PASSWORD"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "supersecret"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_preview", "false"),
					resource.TestCheckResourceAttr("coolify_environment_variable.test", "is_build", "false"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_DatabaseUpdate
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_DatabaseUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentEnvVar := client.EnvironmentVariable{
		UUID: "env-db-upd-uuid", Key: "DB_MAX_CONN", Value: "10", IsPreview: false, IsBuild: false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentEnvVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{currentEnvVar})
	})
	mux.HandleFunc("PATCH /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["value"].(string); ok {
			currentEnvVar.Value = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/databases/{dbUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != currentEnvVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					database_uuid = "dddd0001-0001-4000-8000-000000000001"
					key           = "DB_MAX_CONN"
					value         = "10"
				`),
				Check: resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "10"),
			},
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					database_uuid = "dddd0001-0001-4000-8000-000000000001"
					key           = "DB_MAX_CONN"
					value         = "50"
				`),
				Check: resource.TestCheckResourceAttr("coolify_environment_variable.test", "value", "50"),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvironmentVariableResource_DatabaseImport
// ---------------------------------------------------------------------------

func TestEnvironmentVariableResource_DatabaseBuildVarRejected(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig("https://example.invalid", `
					database_uuid = "dddd0001-0001-4000-8000-000000000001"
					key           = "DB_MAX_CONN"
					value         = "50"
					is_build      = true
				`),
				ExpectError: regexp.MustCompile("application-scoped environment variables"),
			},
		},
	})
}

func TestEnvironmentVariableResource_DatabaseImport(t *testing.T) {
	t.Parallel()
	envVar := client.EnvironmentVariable{
		UUID: "eeee0003-0003-4000-8000-000000000001", Key: "DB_VAR", Value: "db-value", IsPreview: false, IsBuild: false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": envVar.UUID})
	})
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/envs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.EnvironmentVariable{envVar})
	})
	mux.HandleFunc("DELETE /api/v1/databases/{dbUUID}/envs/{envUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" || r.PathValue("envUUID") != envVar.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testEnvVarResourceConfig(srv.URL, `
					database_uuid = "dddd0001-0001-4000-8000-000000000001"
					key           = "DB_VAR"
					value         = "db-value"
				`),
			},
			{
				ResourceName:                         "coolify_environment_variable.test",
				ImportState:                          true,
				ImportStateId:                        "database:dddd0001-0001-4000-8000-000000000001:eeee0003-0003-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"value"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testEnvVarResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_environment_variable", "test", attrs)
}

func checkEnvVarDestroy(serverURL, parentType, parentUUID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "coolify_environment_variable" {
				continue
			}
			uuid := rs.Primary.Attributes["uuid"]
			if uuid == "" {
				continue
			}
			resp, err := http.Get(serverURL + "/api/v1/" + parentType + "/" + parentUUID + "/envs")
			if err != nil {
				return fmt.Errorf("checking env var destroy: %w", err)
			}
			defer resp.Body.Close()
			var envVars []client.EnvironmentVariable
			if err := json.NewDecoder(resp.Body).Decode(&envVars); err != nil {
				return fmt.Errorf("decoding env var list: %w", err)
			}
			for _, ev := range envVars {
				if ev.UUID == uuid {
					return fmt.Errorf("coolify_environment_variable %s still exists", uuid)
				}
			}
		}
		return nil
	}
}
