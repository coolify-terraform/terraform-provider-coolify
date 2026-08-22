package instanceemail_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestInstanceEmailSettingsDataSource_Read(t *testing.T) {
	t.Parallel()
	store := &mockInstanceEmail{
		SMTPEnabled:    true,
		SMTPHost:       "smtp.example.com",
		SMTPPort:       587,
		SMTPEncryption: "starttls",
		SMTPEhloDomain: "mail.example.com",
	}
	srv := newMockServer(store)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_instance_email_settings" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "id", "current"),
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "smtp_enabled", "true"),
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "smtp_host", "smtp.example.com"),
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "smtp_port", "587"),
					resource.TestCheckResourceAttr("data.coolify_instance_email_settings.test", "smtp_ehlo_domain", "mail.example.com"),
				),
			},
		},
	})
}

func TestInstanceEmailSettingsDataSource_ReadAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_instance_email_settings" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error reading instance email settings`),
			},
		},
	})
}

func TestInstanceEmailSettingsDataSource_ReadNotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.10"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_instance_email_settings" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error reading instance email settings`),
			},
		},
	})
}

func TestInstanceEmailSettingsDataSource_ReadUnsupportedVersion(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.9"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_instance_email_settings" "test" {}
`,
				ExpectError: regexp.MustCompile(`Coolify version cannot read instance email settings`),
			},
		},
	})
}
