package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	servicepkg "github.com/coolify-terraform/terraform-provider-coolify/internal/service/service"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// expectedWritableServiceUpdateKeys lists the JSON keys that UpdateServiceInput
// must expose. These match the 8 fields accepted by
// ServicesController::update_by_uuid in the Coolify API.
var expectedWritableServiceUpdateKeys = []string{
	"connect_to_docker_network",
	"description",
	"docker_compose_raw",
	"force_domain_override",
	"instant_deploy",
	"is_container_label_escape_enabled",
	"name",
	"urls",
}

func TestUpdateServiceInput_PublicPatchSurfaceMatchesExpectedKeys(t *testing.T) {
	t.Parallel()
	updateType := reflect.TypeOf(client.UpdateServiceInput{})
	actualKeys := make([]string, 0, updateType.NumField())
	for i := 0; i < updateType.NumField(); i++ {
		key, _, _ := strings.Cut(updateType.Field(i).Tag.Get("json"), ",")
		if key == "" || key == "-" {
			continue
		}
		actualKeys = append(actualKeys, key)
	}
	sort.Strings(actualKeys)
	if !reflect.DeepEqual(actualKeys, expectedWritableServiceUpdateKeys) {
		t.Fatalf("UpdateServiceInput PATCH keys = %v, want %v", actualKeys, expectedWritableServiceUpdateKeys)
	}
}

const serviceTestConfig = `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"
}
`

func serviceConfig(serverURL string) string {
	return acctest.ProviderBlockForURL(serverURL) + serviceTestConfig
}

// serviceURLEntry mirrors the service URL model shape for test state structs.
type serviceURLEntry struct {
	Name types.String `tfsdk:"name"`
	URL  types.String `tfsdk:"url"`
}

type mockServiceState struct {
	mu          sync.Mutex
	uuid        string
	name        string
	description string
	deleted     bool
}

func newMockServiceServer() (*httptest.Server, *mockServiceState) {
	state := &mockServiceState{
		uuid: "dddd0001-0001-4000-8000-000000000001",
		name: "plausible-svc",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			for _, field := range []string{"project_uuid", "server_uuid"} {
				if _, ok := body[field]; !ok {
					http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
					return
				}
			}
			// Must have either type or docker_compose_raw
			_, hasType := body["type"]
			_, hasCompose := body["docker_compose_raw"]
			if !hasType && !hasCompose {
				http.Error(w, `{"error":"one of type or docker_compose_raw is required"}`, http.StatusUnprocessableEntity)
				return
			}
			if hasType && hasCompose {
				http.Error(w, `{"message":"You cannot provide both service type and docker_compose_raw."}`, http.StatusUnprocessableEntity)
				return
			}
			if v, ok := body["name"].(string); ok && v != "" {
				state.name = v
			}
			if v, ok := body["description"].(string); ok {
				state.description = v
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             state.uuid,
				"name":             state.name,
				"description":      state.description,
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"type":             "plausible",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			state.deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	return srv, state
}

func TestServiceResource_CreateImport(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{
			// Create
			{
				Config: serviceConfig(srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "dddd0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_service.test", "name", "plausible-svc"),
					resource.TestCheckResourceAttr("coolify_service.test", "type", "plausible"),
					resource.TestCheckResourceAttr("coolify_service.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_service.test", "instant_deploy", "false"),
				),
			},
			// Idempotency
			{
				Config:             serviceConfig(srv.URL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Import
			{
				ResourceName:      "coolify_service.test",
				ImportState:       true,
				ImportStateId:     "dddd0001-0001-4000-8000-000000000001",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{"project_uuid", "server_uuid", "environment_name", "type"},
			},
		},
	})
}

func TestServiceResource_ImportBadSimpleUUID(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(srv.URL),
			},
			{
				ResourceName:  "coolify_service.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestServiceResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	state := &mockServiceState{uuid: "dddd0009-0009-4000-8000-000000000009", name: "plausible-svc"}
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			forceReadFailure.Store(true)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             state.uuid,
				"name":             state.name,
				"description":      state.description,
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"type":             "plausible",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			state.deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      serviceConfig(srv.URL),
				ExpectError: regexp.MustCompile(`(?s)Service created but refresh failed.*Could not read service.*partial Terraform state was saved`),
			},
		},
	})
}

func TestDeleteService_AddsWarningWhenPollingTimesOut(t *testing.T) {
	t.Parallel()

	const uuid = "svc-delete-timeout-uuid"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", uuid):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", uuid):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"` + uuid + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"test"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	// Use a timeout long enough for the DELETE request to succeed but short
	// enough that PollUntilDeleted gives up before the resource disappears
	// (the mock always returns 200, so the resource never actually goes away).
	// The previous 10ms timeout caused flakes under concurrent test load
	// because the retryablehttp client couldn't complete even the DELETE.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res := servicepkg.NewResource()
	cfgRes, ok := res.(fwresource.ResourceWithConfigure)
	if !ok {
		t.Fatal("service resource does not implement ResourceWithConfigure")
	}
	var cfgResp fwresource.ConfigureResponse
	cfgRes.Configure(ctx, fwresource.ConfigureRequest{ProviderData: c}, &cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure errors: %v", cfgResp.Diagnostics.Errors())
	}

	var schemaResp fwresource.SchemaResponse
	res.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", schemaResp.Diagnostics.Errors())
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	setDiags := state.Set(ctx, struct {
		Timeouts                      timeouts.Value    `tfsdk:"timeouts"`
		UUID                          types.String      `tfsdk:"uuid"`
		Name                          types.String      `tfsdk:"name"`
		Description                   types.String      `tfsdk:"description"`
		ProjectUUID                   types.String      `tfsdk:"project_uuid"`
		ServerUUID                    types.String      `tfsdk:"server_uuid"`
		EnvironmentName               types.String      `tfsdk:"environment_name"`
		Type                          types.String      `tfsdk:"type"`
		Status                        types.String      `tfsdk:"status"`
		DockerCompose                 types.String      `tfsdk:"docker_compose"`
		DockerComposeRaw              types.String      `tfsdk:"docker_compose_raw"`
		ConnectToNetwork              types.Bool        `tfsdk:"connect_to_docker_network"`
		IsContainerLabelEscapeEnabled types.Bool        `tfsdk:"is_container_label_escape_enabled"`
		ConfigHash                    types.String      `tfsdk:"config_hash"`
		InstantDeploy                 types.Bool        `tfsdk:"instant_deploy"`
		URLs                          []serviceURLEntry `tfsdk:"urls"`
		ForceDomainOverride           types.Bool        `tfsdk:"force_domain_override"`
	}{
		Timeouts:                      timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{"create": types.StringType})},
		UUID:                          types.StringValue(uuid),
		Name:                          types.StringNull(),
		Description:                   types.StringNull(),
		ProjectUUID:                   types.StringNull(),
		ServerUUID:                    types.StringNull(),
		EnvironmentName:               types.StringNull(),
		Type:                          types.StringNull(),
		Status:                        types.StringNull(),
		DockerCompose:                 types.StringNull(),
		DockerComposeRaw:              types.StringNull(),
		ConnectToNetwork:              types.BoolNull(),
		IsContainerLabelEscapeEnabled: types.BoolNull(),
		ConfigHash:                    types.StringNull(),
		InstantDeploy:                 types.BoolNull(),
		URLs:                          nil,
		ForceDomainOverride:           types.BoolNull(),
	})
	if setDiags.HasError() {
		t.Fatalf("unexpected state set errors: %v", setDiags.Errors())
	}

	resp := &fwresource.DeleteResponse{State: state}
	res.Delete(ctx, fwresource.DeleteRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected 1 warning, got %d", resp.Diagnostics.WarningsCount())
	}
	warning := resp.Diagnostics.Warnings()[0]
	if warning.Summary() != "Delete is still finishing in Coolify" {
		t.Fatalf("warning summary = %q, want %q", warning.Summary(), "Delete is still finishing in Coolify")
	}
	if !strings.Contains(warning.Detail(), uuid) {
		t.Fatalf("warning detail %q does not mention uuid %s", warning.Detail(), uuid)
	}
	if !strings.Contains(warning.Detail(), "may still exist temporarily") {
		t.Fatalf("warning detail %q does not explain the temporary remote state", warning.Detail())
	}
}

func TestServiceResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	svcUUID := "svc-disappear-uuid-001"

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": svcUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", svcUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             svcUUID,
				"name":             "disappearing-svc",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"type":             "plausible",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", svcUUID):
			deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		case strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_service.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_service.test", "/api/v1/services/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestServiceResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentDesc := "initial description"
	deleted := false

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": "svc-uuid-1"})

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["description"].(string); ok {
				currentDesc = v
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             "svc-uuid-1",
				"name":             "plausible-svc",
				"description":      currentDesc,
				"type":             "plausible",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
			})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/services/svc-uuid-"):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             "svc-uuid-1",
				"name":             "plausible-svc",
				"description":      currentDesc,
				"type":             "plausible",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
			})

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	baseConfig := func(desc string) string {
		return acctest.ProviderBlockForURL(srv.URL) + fmt.Sprintf(`
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"
  description  = %q
}
`, desc)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: baseConfig("initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-uuid-1"),
					resource.TestCheckResourceAttr("coolify_service.test", "description", "initial description"),
				),
			},
			{
				Config: baseConfig("updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// UUID stays the same, proving in-place update (no destroy+recreate).
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-uuid-1"),
					resource.TestCheckResourceAttr("coolify_service.test", "description", "updated description"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_UpdateBoolFields
// ---------------------------------------------------------------------------

func TestServiceResource_UpdateBoolFields(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	labelEscape := false
	deleted := false

	svcResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":                              "svc-bool-uuid-1",
			"name":                              "plausible-svc",
			"type":                              "plausible",
			"project_uuid":                      "aaaa0001-0001-4000-8000-000000000001",
			"server_uuid":                       "bbbb0001-0001-4000-8000-000000000001",
			"environment_name":                  "production",
			"is_container_label_escape_enabled": labelEscape,
		}
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": "svc-bool-uuid-1"})

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["is_container_label_escape_enabled"].(bool); ok {
				labelEscape = v
			}
			json.NewEncoder(w).Encode(svcResponse())

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/services/svc-bool-uuid-"):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(svcResponse())

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	configFn := func(escape bool) string {
		return acctest.ProviderBlockForURL(srv.URL) + fmt.Sprintf(`
resource "coolify_service" "test" {
  project_uuid                       = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid                        = "bbbb0001-0001-4000-8000-000000000001"
  type                               = "plausible"
  is_container_label_escape_enabled  = %t
}
`, escape)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: configFn(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-bool-uuid-1"),
					resource.TestCheckResourceAttr("coolify_service.test", "is_container_label_escape_enabled", "false"),
				),
			},
			{
				Config: configFn(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "svc-bool-uuid-1"),
					resource.TestCheckResourceAttr("coolify_service.test", "is_container_label_escape_enabled", "true"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_ImportCompound
// ---------------------------------------------------------------------------

func TestServiceResource_ImportCompound(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	const (
		projUUID = "aaaa0001-0001-4000-8000-000000000001"
		srvUUID  = "bbbb0001-0001-4000-8000-000000000001"
		svcUUID  = "dddd0001-0001-4000-8000-000000000001"
		envName  = "production"
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(srv.URL),
			},
			{
				ResourceName:  "coolify_service.test",
				ImportState:   true,
				ImportStateId: projUUID + ":" + srvUUID + ":" + envName + ":" + svcUUID,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(states))
					}
					attrs := states[0].Attributes
					checks := map[string]string{
						"project_uuid":     projUUID,
						"server_uuid":      srvUUID,
						"environment_name": envName,
						"uuid":             svcUUID,
					}
					for k, want := range checks {
						if got := attrs[k]; got != want {
							return fmt.Errorf("attribute %s = %q, want %q", k, got, want)
						}
					}
					return nil
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_ImportCompoundBadParts
// ---------------------------------------------------------------------------

func TestServiceResource_ImportCompoundBadParts(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(srv.URL),
			},
			{
				ResourceName:  "coolify_service.test",
				ImportState:   true,
				ImportStateId: "a:b:c",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_ImportCompoundEmptyEnv
// ---------------------------------------------------------------------------

func TestServiceResource_ImportCompoundEmptyEnv(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(srv.URL),
			},
			{
				ResourceName:  "coolify_service.test",
				ImportState:   true,
				ImportStateId: "aaaa0001-0001-4000-8000-000000000001:bbbb0001-0001-4000-8000-000000000001::dddd0001-0001-4000-8000-000000000001",
				ExpectError:   regexp.MustCompile(`environment_name must not be empty`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_MutualExclusivity_TypeAndCompose
// ---------------------------------------------------------------------------

func TestServiceResource_MutualExclusivity_TypeAndCompose(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid       = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid        = "bbbb0001-0001-4000-8000-000000000001"
  type               = "plausible"
  docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
}
`,
				ExpectError: regexp.MustCompile(`(?i)mutually exclusive|conflicting`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_MissingTypeAndCompose
// ---------------------------------------------------------------------------

func TestServiceResource_MissingTypeAndCompose(t *testing.T) {
	t.Parallel()
	srv, _ := newMockServiceServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`(?i)must be set|missing required`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_ComposeCreate
// ---------------------------------------------------------------------------

func newComposeAwareMockServer() (*httptest.Server, *mockServiceState) {
	state := &mockServiceState{
		uuid: "dddd0002-0002-4000-8000-000000000002",
		name: "custom-compose-svc",
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"bad body"}`, http.StatusBadRequest)
				return
			}
			// Verify docker_compose_raw was sent (not type)
			if _, ok := body["docker_compose_raw"]; !ok {
				http.Error(w, `{"error":"docker_compose_raw required"}`, http.StatusUnprocessableEntity)
				return
			}
			if v, ok := body["name"].(string); ok && v != "" {
				state.name = v
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": state.uuid})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":               state.uuid,
				"name":               state.name,
				"project_uuid":       "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":        "bbbb0001-0001-4000-8000-000000000001",
				"environment_name":   "production",
				"type":               "custom",
				"docker_compose_raw": "version: '3'\nservices:\n  web:\n    image: nginx\n",
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", state.uuid):
			state.deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	return srv, state
}

func TestServiceResource_ComposeCreate(t *testing.T) {
	t.Parallel()
	srv, _ := newComposeAwareMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid       = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid        = "bbbb0001-0001-4000-8000-000000000001"
  docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", "dddd0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("coolify_service.test", "name", "custom-compose-svc"),
				),
			},
			// Idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid       = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid        = "bbbb0001-0001-4000-8000-000000000001"
  docker_compose_raw = "version: '3'\nservices:\n  web:\n    image: nginx\n"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestServiceResource_WithURLs
// ---------------------------------------------------------------------------

func TestServiceResource_WithURLs(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	svcUUID := "svc-urls-uuid-001"
	var lastURLs []map[string]interface{}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if urls, ok := body["urls"].([]interface{}); ok {
				lastURLs = nil
				for _, u := range urls {
					if m, ok := u.(map[string]interface{}); ok {
						lastURLs = append(lastURLs, m)
					}
				}
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"uuid": svcUUID})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", svcUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			// Return applications with fqdn to simulate the read-back.
			apps := []map[string]interface{}{}
			for _, u := range lastURLs {
				apps = append(apps, map[string]interface{}{
					"name": u["name"],
					"fqdn": u["url"],
				})
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             svcUUID,
				"name":             "plausible-svc",
				"type":             "plausible",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"applications":     apps,
			})

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/services/"):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if urls, ok := body["urls"].([]interface{}); ok {
				lastURLs = nil
				for _, u := range urls {
					if m, ok := u.(map[string]interface{}); ok {
						lastURLs = append(lastURLs, m)
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"uuid": svcUUID})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/services/%s", svcUUID):
			deleted = true
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"

  urls = [{
    name = "web"
    url  = "https://app.example.com"
  }]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "uuid", svcUUID),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.#", "1"),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.0.name", "web"),
					resource.TestCheckResourceAttr("coolify_service.test", "urls.0.url", "https://app.example.com"),
				),
			},
			// Idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"

  urls = [{
    name = "web"
    url  = "https://app.example.com"
  }]
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
