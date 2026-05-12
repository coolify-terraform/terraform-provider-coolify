package provider_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProvider_MissingToken(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://localhost:8000"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Missing Coolify Token`),
			},
		},
	})
}

func TestProvider_MissingEndpoint(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  token = "some-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Missing Coolify Endpoint`),
			},
		},
	})
}

func TestProvider_UnreachableEndpoint(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://127.0.0.1:1"
  token    = "bad-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Unable to connect to Coolify`),
			},
		},
	})
}

func TestProvider_InvalidToken(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			http.Error(w, `{"message":"Unauthenticated."}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Unable to connect to Coolify`),
			},
		},
	})
}

func TestProvider_InvalidEndpointScheme(t *testing.T) {
	t.Setenv("COOLIFY_TOKEN", "")
	t.Setenv("COOLIFY_ENDPOINT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "ftp://localhost:8000"
  token    = "test-token"
}

data "coolify_team" "test" {
  id = 0
}
`,
				ExpectError: regexp.MustCompile(`Invalid Coolify Endpoint`),
			},
		},
	})
}
