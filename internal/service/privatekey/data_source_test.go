package privatekey_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPrivateKeyDataSource(t *testing.T) {
	key := &client.PrivateKey{
		UUID:         "pk-ds-uuid-1",
		Name:         "data-source-key",
		Description:  "A key for testing",
		PrivateKey:   "ssh-ed25519 AAAA-ds-test",
		IsGitRelated: true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/") {
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			if uuid == key.UUID {
				json.NewEncoder(w).Encode(key)
				return
			}
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
data "coolify_private_key" "test" {
  uuid = "pk-ds-uuid-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_private_key.test", "uuid", "pk-ds-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_private_key.test", "name", "data-source-key"),
					resource.TestCheckResourceAttr("data.coolify_private_key.test", "description", "A key for testing"),
					resource.TestCheckResourceAttr("data.coolify_private_key.test", "private_key", "ssh-ed25519 AAAA-ds-test"),
					resource.TestCheckResourceAttr("data.coolify_private_key.test", "is_git_related", "true"),
				),
			},
		},
	})
}
