package notificationemail_test

import (
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

type mockEmail struct {
	mu               sync.Mutex
	SMTPEnabled      bool
	SMTPHost         string
	SMTPPort         int
	SMTPEncryption   string
	ResendEnabled    bool
	UseInstanceEmail bool
	notificationcommon.EventStore
}

func (m *mockEmail) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id":                          1,
		"team_id":                     0,
		"smtp_enabled":                m.SMTPEnabled,
		"smtp_host":                   m.SMTPHost,
		"smtp_encryption":             m.SMTPEncryption,
		"resend_enabled":              m.ResendEnabled,
		"use_instance_email_settings": m.UseInstanceEmail,
	}
	if m.SMTPPort != 0 {
		out["smtp_port"] = m.SMTPPort
	}
	m.PutSnapshot(out, "email")
	return out
}

func newMockServer(store *mockEmail) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		allowed := notificationcommon.MergeAllowed(
			notificationcommon.EventAllowedFields("email"),
			"smtp_enabled", "smtp_from_address", "smtp_from_name",
			"smtp_recipients", "smtp_host", "smtp_port",
			"smtp_encryption", "smtp_username", "smtp_password",
			"smtp_timeout", "resend_enabled", "resend_api_key",
			"use_instance_email_settings",
		)
		if notificationcommon.RejectUnknownFields(w, body, allowed) {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "smtp_enabled", &store.SMTPEnabled)
		notificationcommon.StringFromBody(body, "smtp_host", &store.SMTPHost)
		notificationcommon.StringFromBody(body, "smtp_encryption", &store.SMTPEncryption)
		if v, ok := body["smtp_port"].(float64); ok {
			store.SMTPPort = int(v)
		}
		notificationcommon.BoolFromBody(body, "resend_enabled", &store.ResendEnabled)
		notificationcommon.BoolFromBody(body, "use_instance_email_settings", &store.UseInstanceEmail)
		store.ApplyBody("email", body)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestEmailNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockEmail{EventStore: notificationcommon.EventStore{DeploymentFailure: true}}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled       = true
  smtp_host          = "smtp.example.com"
  smtp_port          = 587
  smtp_encryption    = "starttls"
  deployment_failure = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "id", "current"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_host", "smtp.example.com"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_port", "587"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_encryption", "starttls"),
					func(_ *terraform.State) error {
						store.mu.Lock()
						defer store.mu.Unlock()
						if !store.SMTPEnabled || store.SMTPHost != "smtp.example.com" || store.SMTPPort != 587 {
							t.Fatalf("store %+v", store)
						}
						return nil
					},
				),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled       = true
  smtp_host          = "smtp.example.com"
  smtp_port          = 465
  smtp_encryption    = "tls"
  deployment_failure = true
  backup_failure     = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_port", "465"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_encryption", "tls"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "backup_failure", "true"),
				),
			},
			{
				ResourceName:            "coolify_notification_email.test",
				ImportState:             true,
				ImportStateId:           "current",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"smtp_password", "smtp_username", "smtp_from_address", "smtp_from_name", "smtp_recipients", "resend_api_key"},
			},
		},
	})
}

func TestEmailNotificationResource_InvalidEncryption(t *testing.T) {
	t.Parallel()
	store := &mockEmail{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled    = true
  smtp_encryption = "ssl"
`),
				ExpectError: regexp.MustCompile(`starttls|tls|none`),
			},
		},
	})
}

func TestEmailNotificationResource_DestroyDisablesAll(t *testing.T) {
	t.Parallel()
	store := &mockEmail{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.SMTPEnabled || store.ResendEnabled || store.UseInstanceEmail {
				return fmt.Errorf("expected all enables false after destroy: smtp=%v resend=%v instance=%v",
					store.SMTPEnabled, store.ResendEnabled, store.UseInstanceEmail)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled                = true
  resend_enabled              = true
  use_instance_email_settings = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "resend_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "use_instance_email_settings", "true"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL),
			},
		},
	})
}

func TestEmailNotificationResource_PreserveHiddenSecrets(t *testing.T) {
	t.Parallel()
	store := &mockEmail{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		notificationcommon.WriteJSON(w, map[string]interface{}{
			"id": 1, "team_id": 0,
			"smtp_enabled": true,
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
		body, ok := notificationcommon.DecodeJSONBody(w, r)
		if !ok {
			return
		}
		store.mu.Lock()
		notificationcommon.BoolFromBody(body, "smtp_enabled", &store.SMTPEnabled)
		notificationcommon.StringFromBody(body, "smtp_host", &store.SMTPHost)
		store.mu.Unlock()
		notificationcommon.WriteJSON(w, map[string]interface{}{
			"id": 1, "team_id": 0, "smtp_enabled": true,
		})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled  = true
  smtp_host     = "smtp.example.com"
  smtp_password = "s3cret"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_host", "smtp.example.com"),
					resource.TestCheckResourceAttr("coolify_notification_email.test", "smtp_password", "s3cret"),
				),
			},
		},
	})
}

func TestEmailNotificationResource_InvalidImport(t *testing.T) {
	t.Parallel()
	store := &mockEmail{}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled = true
  smtp_host    = "smtp.example.com"
`),
			},
			{
				ResourceName:  "coolify_notification_email.test",
				ImportState:   true,
				ImportStateId: "not-current",
				ExpectError:   regexp.MustCompile(`team singleton|import with id "current"`),
			},
		},
	})
}

func TestEmailNotificationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":false}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled = true
  smtp_host    = "smtp.example.com"
`),
				ExpectError: regexp.MustCompile(`Error configuring email notifications`),
			},
		},
	})
}

func TestEmailNotificationResource_ReadAPIError(t *testing.T) {
	t.Parallel()
	var getCount int
	var patchDone bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		getCount++
		// Post-create refresh is get #1 after patch; fail subsequent Reads (step 2).
		if patchDone && getCount >= 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":true,"smtp_host":"smtp.example.com"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		patchDone = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":true,"smtp_host":"smtp.example.com"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled = true
  smtp_host    = "smtp.example.com"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled = true
  smtp_host    = "smtp.example.com"
`),
				ExpectError: regexp.MustCompile(`Error reading email notifications`),
			},
		},
	})
}

func TestEmailNotificationResource_UpdateAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
  deployment_failure = true
`),
				ExpectError: regexp.MustCompile(`Error updating email notifications`),
			},
		},
	})
}

func TestEmailNotificationResource_DestroyAPIError(t *testing.T) {
	t.Parallel()
	var patches int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		patches++
		// Create succeeds (patch 1); destroy disable fails (patch 2+).
		if patches == 2 {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"team_id":0,"smtp_enabled":true,"smtp_host":"smtp.example.com","smtp_port":587,"smtp_encryption":"starttls"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_notification_email", "test", `
  smtp_enabled    = true
  smtp_host       = "smtp.example.com"
  smtp_port       = 587
  smtp_encryption = "starttls"
`),
			},
			{
				Config:      acctest.ProviderBlockForURL(srv.URL),
				ExpectError: regexp.MustCompile(`Error disabling email notifications on destroy`),
			},
		},
	})
}
