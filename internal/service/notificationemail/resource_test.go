package notificationemail_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockEmail struct {
	mu                sync.Mutex
	SMTPEnabled       bool
	SMTPHost          string
	SMTPPort          int
	SMTPEncryption    string
	ResendEnabled     bool
	UseInstanceEmail  bool
	DeploymentFailure bool
	BackupFailure     bool
}

func (m *mockEmail) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]interface{}{
		"id": 1, "team_id": 0,
		"smtp_enabled":                           m.SMTPEnabled,
		"smtp_host":                              m.SMTPHost,
		"smtp_encryption":                        m.SMTPEncryption,
		"resend_enabled":                         m.ResendEnabled,
		"use_instance_email_settings":            m.UseInstanceEmail,
		"deployment_failure_email_notifications": m.DeploymentFailure,
		"backup_failure_email_notifications":     m.BackupFailure,
	}
	if m.SMTPPort != 0 {
		out["smtp_port"] = m.SMTPPort
	}
	return out
}

func newMockServer(store *mockEmail) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"message":"bad json"}`, http.StatusBadRequest)
			return
		}
		// Reject unknown fields like Coolify.
		allowed := map[string]bool{
			"smtp_enabled": true, "smtp_from_address": true, "smtp_from_name": true,
			"smtp_recipients": true, "smtp_host": true, "smtp_port": true,
			"smtp_encryption": true, "smtp_username": true, "smtp_password": true,
			"smtp_timeout": true, "resend_enabled": true, "resend_api_key": true,
			"use_instance_email_settings":            true,
			"deployment_success_email_notifications": true, "deployment_failure_email_notifications": true,
			"status_change_email_notifications": true, "backup_success_email_notifications": true,
			"backup_failure_email_notifications": true, "scheduled_task_success_email_notifications": true,
			"scheduled_task_failure_email_notifications": true, "docker_cleanup_success_email_notifications": true,
			"docker_cleanup_failure_email_notifications": true, "server_disk_usage_email_notifications": true,
			"server_reachable_email_notifications": true, "server_unreachable_email_notifications": true,
			"server_patch_email_notifications": true, "traefik_outdated_email_notifications": true,
		}
		for k := range body {
			if !allowed[k] {
				http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
				return
			}
		}
		store.mu.Lock()
		if v, ok := body["smtp_enabled"].(bool); ok {
			store.SMTPEnabled = v
		}
		if v, ok := body["smtp_host"].(string); ok {
			store.SMTPHost = v
		}
		if v, ok := body["smtp_encryption"].(string); ok {
			store.SMTPEncryption = v
		}
		if v, ok := body["smtp_port"].(float64); ok {
			store.SMTPPort = int(v)
		}
		if v, ok := body["resend_enabled"].(bool); ok {
			store.ResendEnabled = v
		}
		if v, ok := body["use_instance_email_settings"].(bool); ok {
			store.UseInstanceEmail = v
		}
		if v, ok := body["deployment_failure_email_notifications"].(bool); ok {
			store.DeploymentFailure = v
		}
		if v, ok := body["backup_failure_email_notifications"].(bool); ok {
			store.BackupFailure = v
		}
		store.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	return httptest.NewServer(acctest.WithVersionEndpoint(mux))
}

func TestEmailNotificationResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	store := &mockEmail{DeploymentFailure: true}
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "team_id": 0,
			"smtp_enabled": true,
		})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		store.mu.Lock()
		if v, ok := body["smtp_enabled"].(bool); ok {
			store.SMTPEnabled = v
		}
		if v, ok := body["smtp_host"].(string); ok {
			store.SMTPHost = v
		}
		store.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
