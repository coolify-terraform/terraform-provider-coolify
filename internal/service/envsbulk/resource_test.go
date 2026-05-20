package envsbulk_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvsBulkResource_Create(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs/bulk", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		_ = json.Unmarshal(body, &payload)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/applications/550e8400-e29b-41d4-a716-446655440010/envs", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"uuid":"env-1","key":"APP_ENV","value":"production","is_preview":false,"is_buildtime":false},{"uuid":"env-2","key":"LOG_LEVEL","value":"info","is_preview":false,"is_buildtime":false}]`))
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
				ImportStateId:                        "service/550e8400-e29b-41d4-a716-446655440012",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "resource_uuid",
			},
		},
	})
}
