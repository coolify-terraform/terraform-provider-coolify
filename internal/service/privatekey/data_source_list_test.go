package privatekey_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPrivateKeysListDataSource(t *testing.T) {
	t.Parallel()
	keys := []client.PrivateKey{
		{
			UUID:         "pk-list-uuid-1",
			Name:         "key-alpha",
			Description:  "First key",
			PublicKey:    "ssh-ed25519 AAAA-alpha",
			Fingerprint:  "SHA256:alpha-fingerprint",
			IsGitRelated: true,
		},
		{
			UUID:         "pk-list-uuid-2",
			Name:         "key-beta",
			Description:  "Second key",
			PublicKey:    "ssh-ed25519 AAAA-beta",
			Fingerprint:  "SHA256:beta-fingerprint",
			IsGitRelated: false,
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/security/keys" {
			json.NewEncoder(w).Encode(keys)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_private_keys" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.0.uuid", "pk-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.0.name", "key-alpha"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.0.description", "First key"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.0.public_key", "ssh-ed25519 AAAA-alpha"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.0.fingerprint", "SHA256:alpha-fingerprint"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.0.is_git_related", "true"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.1.uuid", "pk-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.1.name", "key-beta"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.1.description", "Second key"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.1.public_key", "ssh-ed25519 AAAA-beta"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.1.fingerprint", "SHA256:beta-fingerprint"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.test", "private_keys.1.is_git_related", "false"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_private_keys" "filtered" {
  filter {
    name   = "name"
    values = ["key-beta"]
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_private_keys.filtered", "private_keys.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.filtered", "private_keys.0.name", "key-beta"),
					resource.TestCheckResourceAttr("data.coolify_private_keys.filtered", "private_keys.0.is_git_related", "false"),
				),
			},
		},
	})
}

func TestPrivateKeysListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_private_keys" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing private keys`),
			},
		},
	})
}
