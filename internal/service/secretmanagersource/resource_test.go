package secretmanagersource_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	appUUID    = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	appTwoUUID = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	svcUUID    = "ffffffff-ffff-4fff-8fff-ffffffffffff"
	dbUUID     = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	infToken   = "11111111-1111-4111-8111-111111111111"
	vaultToken = "22222222-2222-4222-8222-222222222222"
	saToken    = "33333333-3333-4333-8333-333333333333"
	stToken    = "44444444-4444-4444-8444-444444444444"
)

type sourceMock struct {
	mu           sync.Mutex
	linked       bool
	token        string
	provider     string
	settings     map[string]string
	get404       bool
	delete404    bool
	dropSettings bool
	providers    map[string]string
}

func (m *sourceMock) handler(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/{resource_type}/{uuid}/secret-manager", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, ok := body["token"]; ok {
			t.Errorf("PATCH sent a token secret")
		}
		if _, ok := body["metadata"]; ok || body["available_tokens"] != nil {
			t.Errorf("PATCH sent metadata or available_tokens: %#v", body)
		}
		if len(body) != 2 {
			t.Errorf("PATCH keys = %#v", body)
		}
		token, _ := body["integration_token_uuid"].(string)
		settings, ok := body["settings"].(map[string]any)
		if !ok {
			t.Errorf("settings = %#v", body["settings"])
			http.Error(w, "settings", http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		prov := m.providers[token]
		if prov == "" {
			http.Error(w, `{"message":"Secret manager integration token not found."}`, http.StatusNotFound)
			return
		}
		stored := map[string]string{}
		if !m.dropSettings {
			for k, v := range settings {
				s, _ := v.(string)
				stored[k] = s
			}
		}
		m.linked = true
		m.token = token
		m.provider = prov
		m.settings = stored
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"integration_token_uuid": token,
			"provider":               prov,
			"settings":               stored,
		})
	})
	mux.HandleFunc("GET /api/v1/{resource_type}/{uuid}/secret-manager", func(w http.ResponseWriter, _ *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.get404 {
			http.Error(w, `{"message":"Application not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if !m.linked {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"linked": false, "integration_token_uuid": nil, "provider": nil, "settings": nil,
				"available_tokens": []any{},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"linked": true, "integration_token_uuid": m.token, "provider": m.provider, "settings": m.settings,
			"available_tokens": []map[string]string{{"uuid": m.token, "name": "token", "provider": m.provider}},
		})
	})
	mux.HandleFunc("DELETE /api/v1/{resource_type}/{uuid}/secret-manager", func(w http.ResponseWriter, _ *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.delete404 || !m.linked {
			http.Error(w, `{"message":"The resource has no secret manager source."}`, http.StatusNotFound)
			return
		}
		m.linked = false
		m.settings = nil
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Secret manager source removed."})
	})
	return mux
}

func (m *sourceMock) server(t *testing.T, version string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(m.handler(t), version))
	t.Cleanup(srv.Close)
	return srv
}

func providerConfig(url, body string) string {
	return acctest.ProviderBlockForURL(url) + body
}

func TestSecretManagerSource_InfisicalLifecycle(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{infToken: "infisical", vaultToken: "vault"}}
	srv := m.server(t, "4.6.0")
	inf := fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  infisical_project_id   = "00000000-0000-4000-8000-000000000001"
  infisical_environment  = "prod"
  infisical_secret_path  = "/app"
}
`, appUUID, infToken)
	vault := fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  vault_mount            = "secret"
  vault_path             = "my-app/production"
}
`, appUUID, vaultToken)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, inf),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "integration_provider", "infisical"),
					resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "infisical_secret_path", "/app"),
				),
			},
			{Config: providerConfig(srv.URL, inf), PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: providerConfig(srv.URL, vault),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("coolify_secret_manager_source.test", plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "integration_provider", "vault"),
					resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "vault_path", "my-app/production"),
					resource.TestCheckNoResourceAttr("coolify_secret_manager_source.test", "infisical_project_id"),
				),
			},
			{
				ResourceName:                         "coolify_secret_manager_source.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "application:" + appUUID,
				ImportStateVerifyIdentifierAttribute: "integration_token_uuid",
			},
		},
	})
}

func TestSecretManagerSource_DopplerServiceAccount(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{saToken: "doppler"}}
	srv := m.server(t, "4.6.0")
	cfg := fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  service_uuid           = %q
  integration_token_uuid = %q
  doppler_project        = "website"
  doppler_config         = "production"
}
`, svcUUID, saToken)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, cfg),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "doppler_project", "website"),
					resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "integration_provider", "doppler"),
				),
			},
			{
				ResourceName:                         "coolify_secret_manager_source.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "service:" + svcUUID,
				ImportStateVerifyIdentifierAttribute: "integration_token_uuid",
			},
		},
	})
}

func TestSecretManagerSource_VaultDatabase(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{vaultToken: "vault"}}
	srv := m.server(t, "4.6.0")
	cfg := fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  database_uuid          = %q
  integration_token_uuid = %q
  vault_mount            = "secret"
  vault_path             = "db/prod"
}
`, dbUUID, vaultToken)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: providerConfig(srv.URL, cfg)},
			{
				ResourceName:                         "coolify_secret_manager_source.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "database:" + dbUUID,
				ImportStateVerifyIdentifierAttribute: "integration_token_uuid",
			},
		},
	})
}

func TestSecretManagerSource_DopplerServiceToken(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}}
	srv := m.server(t, "4.6.0")
	cfg := fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, cfg),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_secret_manager_source.test", "integration_provider", "doppler"),
				resource.TestCheckNoResourceAttr("coolify_secret_manager_source.test", "doppler_project"),
				resource.TestCheckNoResourceAttr("coolify_secret_manager_source.test", "doppler_config"),
			),
		}},
	})
}

func TestSecretManagerSource_DroppedDopplerSettings(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}, dropSettings: true}
	srv := m.server(t, "4.6.0")
	cfg := fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  doppler_project        = "website"
  doppler_config         = "production"
}
`, appUUID, stToken)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config:      providerConfig(srv.URL, cfg),
			ExpectError: regexp.MustCompile(`Coolify did not store doppler_project`),
		}},
	})
}

func TestSecretManagerSource_Disappears(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken)),
			ExpectNonEmptyPlan: true,
			Check: func(*terraform.State) error {
				m.mu.Lock()
				m.get404 = true
				m.mu.Unlock()
				return nil
			},
		}},
	})
}

func TestSecretManagerSource_UnlinkedRead(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken)),
			ExpectNonEmptyPlan: true,
			Check: func(*terraform.State) error {
				m.mu.Lock()
				m.linked = false
				m.mu.Unlock()
				return nil
			},
		}},
	})
}

func TestSecretManagerSource_DeleteNotFound(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}, delete404: true}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken))},
			{Config: acctest.ProviderBlockForURL(srv.URL)},
		},
	})
}

func TestSecretManagerSource_ReplaceParent(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}}
	srv := m.server(t, "4.6.0")
	cfg := func(id string) string {
		return providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, id, stToken))
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: cfg(appUUID)},
			{
				Config: cfg(appTwoUUID),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("coolify_secret_manager_source.test", plancheck.ResourceActionReplace),
				}},
			},
		},
	})
}

func TestSecretManagerSource_ImportUnlinked(t *testing.T) {
	t.Parallel()
	m := &sourceMock{}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken)),
			ResourceName:  "coolify_secret_manager_source.test",
			ImportState:   true,
			ImportStateId: "application:" + appUUID,
			ExpectError:   regexp.MustCompile(`(?i)import|not found|no longer exists|linked`),
		}},
	})
}

func TestSecretManagerSource_ImportBareUUID(t *testing.T) {
	t.Parallel()
	m := &sourceMock{providers: map[string]string{stToken: "doppler"}}
	srv := m.server(t, "4.6.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken))},
			{
				ResourceName:  "coolify_secret_manager_source.test",
				ImportState:   true,
				ImportStateId: appUUID,
				ExpectError:   regexp.MustCompile(`application:<uuid>`),
			},
		},
	})
}

func TestSecretManagerSource_RejectsOldCoolify(t *testing.T) {
	t.Parallel()
	m := &sourceMock{}
	srv := m.server(t, "4.5.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerConfig(srv.URL, fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
}
`, appUUID, stToken)),
			ExpectError: regexp.MustCompile(`Coolify >= v4.6.0`),
		}},
	})
}

func TestSecretManagerSource_ValidateConfig(t *testing.T) {
	t.Parallel()
	m := &sourceMock{}
	srv := m.server(t, "4.6.0")
	cases := []struct {
		name string
		body string
		err  string
	}{
		{"two parents", fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  service_uuid           = %q
  integration_token_uuid = %q
}
`, appUUID, svcUUID, stToken), "Exactly one of application_uuid"},
		{"two groups", fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  infisical_project_id   = "p"
  infisical_environment  = "prod"
  vault_mount            = "secret"
  vault_path             = "app"
}
`, appUUID, stToken), "one provider"},
		{"half doppler", fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  doppler_project        = "website"
}
`, appUUID, stToken), "doppler_project and doppler_config"},
		{"whitespace", fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  doppler_project        = " "
  doppler_config         = "production"
}
`, appUUID, stToken), "empty or whitespace"},
		{"secret path alone", fmt.Sprintf(`
resource "coolify_secret_manager_source" "test" {
  application_uuid       = %q
  integration_token_uuid = %q
  infisical_secret_path  = "/app"
}
`, appUUID, stToken), "infisical_secret_path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{{
					Config:      providerConfig(srv.URL, tc.body),
					ExpectError: regexp.MustCompile(tc.err),
				}},
			})
		})
	}
}
