package gitlabapp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/gitlabapp"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestGitLabAppResource_CRUD(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/gitlab-apps":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "corp-gitlab" {
				t.Errorf("POST name: got %v", body["name"])
			}
			if body["html_url"] != "https://gitlab.example.com" {
				t.Errorf("POST html_url: got %v", body["html_url"])
			}
			if body["api_url"] != "https://gitlab.example.com/api/v4" {
				t.Errorf("POST api_url: got %v", body["api_url"])
			}
			if body["group_name"] != "acme" {
				t.Errorf("POST group_name: got %v", body["group_name"])
			}
			if body["client_id"] != "gitlab-app-id" {
				t.Errorf("POST client_id: got %v", body["client_id"])
			}
			body["id"] = float64(7)
			body["uuid"] = "gggg0001-0001-4000-8000-000000000001"
			if body["api_url"] == nil {
				body["api_url"] = body["html_url"].(string) + "/api/v4"
			}
			store["7"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gitlab-apps":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["is_system_wide"]; ok {
				http.Error(w, `{"message":"The is_system_wide field is not allowed."}`, http.StatusUnprocessableEntity)
				return
			}
			if body["name"] != "corp-gitlab-2" {
				t.Errorf("PATCH name: got %v", body["name"])
			}
			if body["html_url"] != "https://gitlab.example.com" {
				t.Errorf("PATCH html_url: got %v", body["html_url"])
			}
			if body["api_url"] != "https://gitlab.example.com/api/v4" {
				t.Errorf("PATCH api_url: got %v", body["api_url"])
			}
			if body["group_name"] != "acme" {
				t.Errorf("PATCH group_name: got %v", body["group_name"])
			}
			if body["client_id"] != "gitlab-app-id" {
				t.Errorf("PATCH client_id: got %v", body["client_id"])
			}
			for _, v := range store {
				for k, val := range body {
					v[k] = val
				}
				_ = json.NewEncoder(w).Encode(v)
				return
			}
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name       = "corp-gitlab"
  html_url   = "https://gitlab.example.com"
  api_url    = "https://gitlab.example.com/api/v4"
  group_name = "acme"
  client_id  = "gitlab-app-id"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "name", "corp-gitlab"),
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "id", "7"),
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "group_name", "acme"),
					resource.TestCheckResourceAttr("coolify_gitlab_app.test", "client_id", "gitlab-app-id"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name       = "corp-gitlab-2"
  html_url   = "https://gitlab.example.com"
  api_url    = "https://gitlab.example.com/api/v4"
  group_name = "acme"
  client_id  = "gitlab-app-id"
}`,
				Check: resource.TestCheckResourceAttr("coolify_gitlab_app.test", "name", "corp-gitlab-2"),
			},
		},
	})
}

func TestGitLabAppResource_SystemWideRequiresReplace(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/gitlab-apps":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["is_system_wide"].(bool); !ok {
				t.Errorf("POST is_system_wide: got %v", body["is_system_wide"])
			}
			body["id"] = float64(7)
			body["uuid"] = "gggg0001-0001-4000-8000-000000000001"
			if body["api_url"] == nil {
				body["api_url"] = body["html_url"].(string) + "/api/v4"
			}
			store["7"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gitlab-apps":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["is_system_wide"]; ok {
				http.Error(w, `{"message":"The is_system_wide field is not allowed."}`, http.StatusUnprocessableEntity)
				return
			}
			http.Error(w, `{"message":"unexpected in-place update"}`, http.StatusUnprocessableEntity)
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name           = "corp-gitlab"
  html_url       = "https://gitlab.example.com"
  is_system_wide = true
}`,
				Check: resource.TestCheckResourceAttr("coolify_gitlab_app.test", "is_system_wide", "true"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name           = "corp-gitlab"
  html_url       = "https://gitlab.example.com"
  is_system_wide = false
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_gitlab_app.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("coolify_gitlab_app.test", "is_system_wide", "false"),
			},
		},
	})
}

func TestGitLabAppResource_UpdateUnknownIDLookupFails(t *testing.T) {
	t.Parallel()
	var patchZero atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/gitlab-apps", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
	})
	mux.HandleFunc("PATCH /api/v1/gitlab-apps/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "0" {
			patchZero.Store(true)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       0,
			"uuid":     "gggg0001-0001-4000-8000-000000000001",
			"name":     "corp-gitlab-2",
			"html_url": "https://gitlab.example.com",
		})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	ctx := context.Background()
	res := gitlabapp.NewResource()
	cfgRes, ok := res.(fwresource.ResourceWithConfigure)
	if !ok {
		t.Fatal("gitlab app resource does not implement ResourceWithConfigure")
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

	type model struct {
		ID           types.Int64  `tfsdk:"id"`
		UUID         types.String `tfsdk:"uuid"`
		Name         types.String `tfsdk:"name"`
		HTMLURL      types.String `tfsdk:"html_url"`
		APIURL       types.String `tfsdk:"api_url"`
		CustomUser   types.String `tfsdk:"custom_user"`
		CustomPort   types.Int64  `tfsdk:"custom_port"`
		GroupName    types.String `tfsdk:"group_name"`
		ClientID     types.String `tfsdk:"client_id"`
		ClientSecret types.String `tfsdk:"client_secret"`
		WebhookToken types.String `tfsdk:"webhook_token"`
		RedirectURI  types.String `tfsdk:"redirect_uri"`
		IsSystemWide types.Bool   `tfsdk:"is_system_wide"`
	}
	values := model{
		ID:           types.Int64Unknown(),
		UUID:         types.StringValue("gggg0001-0001-4000-8000-000000000001"),
		Name:         types.StringValue("corp-gitlab-2"),
		HTMLURL:      types.StringValue("https://gitlab.example.com"),
		APIURL:       types.StringNull(),
		CustomUser:   types.StringNull(),
		CustomPort:   types.Int64Null(),
		GroupName:    types.StringNull(),
		ClientID:     types.StringNull(),
		ClientSecret: types.StringNull(),
		WebhookToken: types.StringNull(),
		RedirectURI:  types.StringNull(),
		IsSystemWide: types.BoolValue(false),
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, values); diags.HasError() {
		t.Fatalf("unexpected plan set errors: %v", diags.Errors())
	}
	stateValues := values
	stateValues.ID = types.Int64Value(0)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, stateValues); diags.HasError() {
		t.Fatalf("unexpected state set errors: %v", diags.Errors())
	}

	resp := &fwresource.UpdateResponse{State: state}
	res.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Update to add an error when id is unknown and UUID lookup 404s")
	}
	errText := resp.Diagnostics.Errors()[0].Detail()
	if !regexp.MustCompile(`(?i)uuid|resolve|id`).MatchString(errText) {
		t.Fatalf("error detail %q does not mention id/uuid resolution", errText)
	}
	if patchZero.Load() {
		t.Fatal("Update PATCHed /gitlab-apps/0 after UUID lookup 404")
	}
}

func TestGitLabAppResource_Disappears(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/gitlab-apps":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			body["id"] = float64(7)
			body["uuid"] = "gggg0001-0001-4000-8000-000000000001"
			if body["api_url"] == nil {
				body["api_url"] = body["html_url"].(string) + "/api/v4"
			}
			store["7"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gitlab-apps":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name     = "corp-gitlab"
  html_url = "https://gitlab.example.com"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_gitlab_app.test", "uuid"),
				acctest.CheckPathDisappears(srv.URL, "/api/v1/gitlab-apps/7"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestGitLabAppResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/gitlab-apps", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name     = "corp-gitlab"
  html_url = "https://gitlab.example.com"
}`,
			ExpectError: regexp.MustCompile(`Error creating GitLab App`),
		}},
	})
}

func newGitLabAppImportServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/gitlab-apps":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "import-gitlab" {
				t.Errorf("POST name: got %v", body["name"])
			}
			if body["html_url"] != "https://gitlab.example.com" {
				t.Errorf("POST html_url: got %v", body["html_url"])
			}
			body["id"] = float64(7)
			body["uuid"] = "gggg0001-0001-4000-8000-000000000001"
			if body["api_url"] == nil {
				body["api_url"] = body["html_url"].(string) + "/api/v4"
			}
			store["7"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gitlab-apps":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
}

func TestGitLabAppResource_Import(t *testing.T) {
	t.Parallel()
	srv := newGitLabAppImportServer(t)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name     = "import-gitlab"
  html_url = "https://gitlab.example.com"
}`,
				Check: resource.TestCheckResourceAttr("coolify_gitlab_app.test", "id", "7"),
			},
			{
				ResourceName:                         "coolify_gitlab_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"client_secret", "webhook_token"},
			},
		},
	})
}

func TestGitLabAppResource_ImportByUUID(t *testing.T) {
	t.Parallel()
	srv := newGitLabAppImportServer(t)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name     = "import-gitlab"
  html_url = "https://gitlab.example.com"
}`,
				Check: resource.TestCheckResourceAttrSet("coolify_gitlab_app.test", "uuid"),
			},
			{
				ResourceName:                         "coolify_gitlab_app.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore:              []string{"client_secret", "webhook_token"},
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_gitlab_app.test", "uuid"),
			},
		},
	})
}

func TestGitLabAppResource_ImportBadID(t *testing.T) {
	t.Parallel()
	srv := newGitLabAppImportServer(t)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_gitlab_app" "test" {
  name     = "import-gitlab"
  html_url = "https://gitlab.example.com"
}`,
			},
			{
				ResourceName:  "coolify_gitlab_app.test",
				ImportState:   true,
				ImportStateId: "not-a-number-or-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}
