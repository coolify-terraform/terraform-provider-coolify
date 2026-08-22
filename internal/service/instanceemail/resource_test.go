package instanceemail_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockInstanceEmail struct {
	mu             sync.Mutex
	SMTPEnabled    bool
	SMTPHost       string
	SMTPPort       int
	SMTPEncryption string
	SMTPEhloDomain string
	ResendEnabled  bool
}

func (m *mockInstanceEmail) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"smtp_enabled":     m.SMTPEnabled,
		"smtp_host":        m.SMTPHost,
		"smtp_encryption":  m.SMTPEncryption,
		"smtp_ehlo_domain": m.SMTPEhloDomain,
		"resend_enabled":   m.ResendEnabled,
	}
	if m.SMTPPort != 0 {
		out["smtp_port"] = m.SMTPPort
	}
	return out
}

func newMockServer(store *mockInstanceEmail) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		allowed := []string{
			"smtp_enabled", "smtp_from_address", "smtp_from_name", "smtp_host",
			"smtp_port", "smtp_encryption", "smtp_username", "smtp_password",
			"smtp_timeout", "smtp_ehlo_domain", "resend_enabled", "resend_api_key",
		}
		if notificationcommon.RejectUnknownFields(w, body, notificationcommon.MergeAllowed(nil, allowed...)) {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "smtp_enabled", &store.SMTPEnabled)
		notificationcommon.StringFromBody(body, "smtp_host", &store.SMTPHost)
		notificationcommon.StringFromBody(body, "smtp_ehlo_domain", &store.SMTPEhloDomain)
		notificationcommon.StringFromBody(body, "smtp_encryption", &store.SMTPEncryption)
		if v, ok := body["smtp_port"].(float64); ok {
			store.SMTPPort = int(v)
		}
		notificationcommon.BoolFromBody(body, "resend_enabled", &store.ResendEnabled)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
}

func TestInstanceEmailSettingsResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockInstanceEmail{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_host", "smtp.example.com"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_port", "587"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_encryption", "starttls"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if !store.SMTPEnabled || store.SMTPHost != "smtp.example.com" || store.SMTPPort != 587 {
							b, _ := json.Marshal(store.snapshot())
							return fmt.Errorf("create did not persist expected SMTP fields: %s", b)
						}
						return nil
					},
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled     = true
  smtp_host        = "smtp.example.com"
  smtp_port        = 465
  smtp_encryption  = "tls"
  smtp_ehlo_domain = "mail.example.com"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_port", "465"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_encryption", "tls"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_ehlo_domain", "mail.example.com"),
				),
			},
			{
				ResourceName:            "coolify_instance_email_settings.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"smtp_from_address", "smtp_from_name", "smtp_username", "smtp_password", "resend_api_key"},
			},
		},
	})
}

func TestInstanceEmailSettingsResource_InvalidFromAddress(t *testing.T) {
	t.Parallel()
	store := &mockInstanceEmail{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled      = true
  smtp_from_address = "not-an-email"
`),
				ExpectError: regexp.MustCompile(`Invalid email address`),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled = true
`),
				ExpectError: regexp.MustCompile(`(?i)configur(e|ing) instance email|Validation failed`),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_DestroyDisables(t *testing.T) {
	t.Parallel()
	store := &mockInstanceEmail{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.SMTPEnabled || store.ResendEnabled {
				return fmt.Errorf("expected smtp and resend false after destroy: smtp=%v resend=%v",
					store.SMTPEnabled, store.ResendEnabled)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled   = true
  resend_enabled = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "smtp_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_instance_email_settings.test", "resend_enabled", "true"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_ReadNotFound(t *testing.T) {
	t.Parallel()
	var gone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		if gone {
			http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				PreConfig: func() { gone = true },
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestInstanceEmailSettingsResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
				ExpectError: regexp.MustCompile(`Error reading instance email settings`),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_UpdateAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.updated.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
				ExpectError: regexp.MustCompile(`Error updating instance email settings`),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_DestroyAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				Config:      acctest.ProviderBlockForURL(srv.URL),
				ExpectError: regexp.MustCompile(`Error disabling instance email settings on destroy`),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_CreateUnsupportedVersion(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.9"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled = true
`),
				ExpectError: regexp.MustCompile(`Coolify version cannot manage instance email settings`),
			},
		},
	})
}

func TestInstanceEmailSettingsResource_DestroyNotFound(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches >= 2 {
			http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_instance_email_settings", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}
