package hetzner_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestHetznerSSHKeysDataSource(t *testing.T) {
	t.Parallel()

	sshKeys := []client.HetznerSSHKey{
		{ID: 1, Name: "my-key", Fingerprint: "00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff"},
		{ID: 2, Name: "deploy-key", Fingerprint: "ff:ee:dd:cc:bb:aa:99:88:77:66:55:44:33:22:11:00"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hetzner/ssh-keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sshKeys)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_ssh_keys" "test" {
  cloud_provider_token_uuid = "test-token-uuid"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_ssh_keys.test", "ssh_keys.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_ssh_keys.test", "ssh_keys.0.id", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_ssh_keys.test", "ssh_keys.0.name", "my-key"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_ssh_keys.test", "ssh_keys.0.fingerprint", "00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_ssh_keys.test", "ssh_keys.1.id", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_ssh_keys.test", "ssh_keys.1.name", "deploy-key"),
				),
			},
		},
	})
}
