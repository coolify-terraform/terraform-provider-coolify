package envsbulk_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestEnvsBulkResource_Create(t *testing.T) {
	t.Parallel()
	var lastPayload []byte
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs/bulk", func(w http.ResponseWriter, r *http.Request) {
		lastPayload, _ = io.ReadAll(r.Body)
		var payload struct {
			Data []struct {
				Key string `json:"key"`
			} `json:"data"`
		}
		if err := json.Unmarshal(lastPayload, &payload); err != nil || len(payload.Data) == 0 {
			http.Error(w, `{"error":"invalid or empty bulk payload"}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs", func(w http.ResponseWriter, _ *http.Request) {
		// Reconstruct GET response from the last PATCH payload (same pattern as Update test).
		var payload struct {
			Data []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"data"`
		}
		if len(lastPayload) > 0 {
			_ = json.Unmarshal(lastPayload, &payload)
		}
		var envs []map[string]interface{}
		for _, e := range payload.Data {
			envs = append(envs, map[string]interface{}{"uuid": "u-" + e.Key, "key": e.Key, "value": e.Value, "is_preview": false, "is_buildtime": false})
		}
		out, _ := json.Marshal(envs)
		_, _ = w.Write(out)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440010"
					variables = {
						APP_ENV   = "production"
						LOG_LEVEL = "info"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "resource_type", "application"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "resource_uuid", "550e8400-e29b-41d4-a716-446655440010"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.APP_ENV", "production"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.LOG_LEVEL", "info"),
				),
			},
		},
	})
}

func TestEnvsBulkResource_Disappears(t *testing.T) {
	t.Parallel()
	var lastPayload []byte
	disappeared := false

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs/bulk", func(w http.ResponseWriter, r *http.Request) {
		lastPayload, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs", func(w http.ResponseWriter, _ *http.Request) {
		if disappeared {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var payload struct {
			Data []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"data"`
		}
		if len(lastPayload) > 0 {
			_ = json.Unmarshal(lastPayload, &payload)
		}
		var envs []map[string]interface{}
		for _, e := range payload.Data {
			envs = append(envs, map[string]interface{}{"uuid": "u-" + e.Key, "key": e.Key, "value": e.Value, "is_preview": false, "is_buildtime": false})
		}
		out, _ := json.Marshal(envs)
		_, _ = w.Write(out)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440010"
					variables = {
						GONE_VAR = "value"
					}
				`),
				Check: func(s *terraform.State) error {
					// Simulate parent resource deletion after Create.
					disappeared = true
					return nil
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestEnvsBulkResource_Update(t *testing.T) {
	t.Parallel()
	var lastPayload []byte
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/databases/550e8400-e29b-41d4-a716-446655440011/envs/bulk", func(w http.ResponseWriter, r *http.Request) {
		lastPayload, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/databases/550e8400-e29b-41d4-a716-446655440011/envs", func(w http.ResponseWriter, _ *http.Request) {
		// Return what was last sent.
		var payload struct {
			Data []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"data"`
		}
		if len(lastPayload) > 0 {
			_ = json.Unmarshal(lastPayload, &payload)
		}
		var envs []map[string]interface{}
		for _, e := range payload.Data {
			envs = append(envs, map[string]interface{}{"uuid": "u-" + e.Key, "key": e.Key, "value": e.Value, "is_preview": false, "is_buildtime": false})
		}
		out, _ := json.Marshal(envs)
		_, _ = w.Write(out)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "database"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440011"
					variables = {
						DB_HOST = "localhost"
					}
				`),
				Check: resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.DB_HOST", "localhost"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "database"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440011"
					variables = {
						DB_HOST = "db.example.com"
						DB_PORT = "5432"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.DB_HOST", "db.example.com"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.DB_PORT", "5432"),
				),
			},
		},
	})
}

func TestEnvsBulkResource_PrefersNonPreviewDuplicateValues(t *testing.T) {
	t.Parallel()
	var lastPayload []byte
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440013/envs/bulk", func(w http.ResponseWriter, r *http.Request) {
		lastPayload, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/applications/550e8400-e29b-41d4-a716-446655440013/envs", func(w http.ResponseWriter, _ *http.Request) {
		if bytes.Contains(lastPayload, []byte(`"ANOTHER"`)) {
			_, _ = w.Write([]byte(`[
				{"uuid":"u-TEST_KEY","key":"TEST_KEY","value":"updated_value","is_preview":false,"is_buildtime":true},
				{"uuid":"u-ANOTHER","key":"ANOTHER","value":"val","is_preview":false,"is_buildtime":true},
				{"uuid":"u-TEST_KEY-preview","key":"TEST_KEY","value":"test_value","is_preview":true,"is_buildtime":true},
				{"uuid":"u-ANOTHER-preview","key":"ANOTHER","value":"val","is_preview":true,"is_buildtime":true}
			]`))
			return
		}
		_, _ = w.Write([]byte(`[
			{"uuid":"u-TEST_KEY","key":"TEST_KEY","value":"test_value","is_preview":false,"is_buildtime":true},
			{"uuid":"u-TEST_KEY-preview","key":"TEST_KEY","value":"test_value","is_preview":true,"is_buildtime":true}
		]`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440013"
					variables = {
						TEST_KEY = "test_value"
					}
				`),
				Check: resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.TEST_KEY", "test_value"),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440013"
					variables = {
						TEST_KEY = "updated_value"
						ANOTHER  = "val"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.TEST_KEY", "updated_value"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.ANOTHER", "val"),
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440013"
					variables = {
						TEST_KEY = "updated_value"
						ANOTHER  = "val"
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestEnvsBulkResource_PreservesSensitiveValuesOnBlankReadBack(t *testing.T) {
	t.Parallel()
	var lastPayload []byte
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440013/envs/bulk", func(w http.ResponseWriter, r *http.Request) {
		lastPayload, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/applications/550e8400-e29b-41d4-a716-446655440013/envs", func(w http.ResponseWriter, _ *http.Request) {
		if len(lastPayload) == 0 {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[
			{"uuid":"u-TEST_KEY","key":"TEST_KEY","value":"","is_preview":false,"is_buildtime":false},
			{"uuid":"u-ANOTHER","key":"ANOTHER","value":"val","is_preview":false,"is_buildtime":false}
		]`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440013"
					variables = {
						TEST_KEY = "updated_value"
						ANOTHER  = "val"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.TEST_KEY", "updated_value"),
					resource.TestCheckResourceAttr("coolify_envs_bulk.test", "variables.ANOTHER", "val"),
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440013"
					variables = {
						TEST_KEY = "updated_value"
						ANOTHER  = "val"
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func newEnvsBulkImportMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/services/550e8400-e29b-41d4-a716-446655440012/envs/bulk", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/services/550e8400-e29b-41d4-a716-446655440012/envs", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"uuid":"e1","key":"REDIS_URL","value":"redis://localhost","is_preview":false,"is_buildtime":false}]`))
	})
	return mux
}

func TestEnvsBulkResource_Import(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/services/550e8400-e29b-41d4-a716-446655440012/envs/bulk", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/services/550e8400-e29b-41d4-a716-446655440012/envs", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"uuid":"e1","key":"REDIS_URL","value":"redis://localhost","is_preview":false,"is_buildtime":false}]`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "service"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440012"
					variables = {
						REDIS_URL = "redis://localhost"
					}
				`),
			},
			{
				ResourceName:                         "coolify_envs_bulk.test",
				ImportState:                          true,
				ImportStateId:                        "service:550e8400-e29b-41d4-a716-446655440012",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "resource_uuid",
			},
		},
	})
}

func envsBulkImportBaseConfig(srvURL string) string {
	return acctest.TestResourceConfig(srvURL, "coolify_envs_bulk", "test", `
		resource_type = "service"
		resource_uuid = "550e8400-e29b-41d4-a716-446655440012"
		variables = {
			REDIS_URL = "redis://localhost"
		}
	`)
}

func TestEnvsBulkResource_ImportBadFormat(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newEnvsBulkImportMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: envsBulkImportBaseConfig(srv.URL)},
			{
				ResourceName:  "coolify_envs_bulk.test",
				ImportState:   true,
				ImportStateId: "bad-format",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestEnvsBulkResource_ImportBadType(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newEnvsBulkImportMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: envsBulkImportBaseConfig(srv.URL)},
			{
				ResourceName:  "coolify_envs_bulk.test",
				ImportState:   true,
				ImportStateId: "invalid/550e8400-e29b-41d4-a716-446655440012",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*must be one of`),
			},
		},
	})
}

func TestEnvsBulkResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newEnvsBulkImportMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: envsBulkImportBaseConfig(srv.URL)},
			{
				ResourceName:  "coolify_envs_bulk.test",
				ImportState:   true,
				ImportStateId: "service:not-a-uuid",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*resource UUID segment`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestEnvsBulkResource_CreateAPIError
// ---------------------------------------------------------------------------

func TestEnvsBulkResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs/bulk", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_envs_bulk", "test", `
					resource_type = "application"
					resource_uuid = "550e8400-e29b-41d4-a716-446655440010"
					variables = {
						APP_ENV = "test"
					}
				`),
				ExpectError: regexp.MustCompile(`Error creating bulk env vars`),
			},
		},
	})
}
