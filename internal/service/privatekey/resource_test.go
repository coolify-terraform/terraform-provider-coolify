package privatekey_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

)

func testProviderBlock(serverURL string) string {
	return `
provider "coolify" {
  endpoint  = "` + serverURL + `"
  token = "test-token"
}`
}

func newPrivateKeyMockServer() *httptest.Server {
	keys := make(map[string]*client.PrivateKey)
	var mu sync.Mutex

	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/security/keys":
			var input client.CreatePrivateKeyInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			key := &client.PrivateKey{
				UUID:         "pk-test-uuid-1",
				Name:         input.Name,
				Description:  input.Description,
				PrivateKey:   input.PrivateKey,
				IsGitRelated: input.IsGitRelated,
			}
			keys[key.UUID] = key
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(key)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			key, ok := keys[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(key)

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			key, ok := keys[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var update client.UpdatePrivateKeyInput
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			if update.Name != nil {
				key.Name = *update.Name
			}
			if update.Description != nil {
				key.Description = *update.Description
			}
			if update.PrivateKey != nil {
				key.PrivateKey = *update.PrivateKey
			}
			if update.IsGitRelated != nil {
				key.IsGitRelated = *update.IsGitRelated
			}
			json.NewEncoder(w).Encode(key)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			delete(keys, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})))
}

func TestPrivateKeyResource_Create(t *testing.T) {
	t.Parallel()
	srv := newPrivateKeyMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "my-ssh-key"
  private_key = "ssh-ed25519 AAAA-test-key"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "pk-test-uuid-1"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", "my-ssh-key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "private_key", "ssh-ed25519 AAAA-test-key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "is_git_related", "false"),
				),
			},
			// Plan idempotency
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "my-ssh-key"
  private_key = "ssh-ed25519 AAAA-test-key"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestPrivateKeyResource_Update(t *testing.T) {
	t.Parallel()
	srv := newPrivateKeyMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "my-ssh-key"
  private_key = "ssh-ed25519 AAAA-test-key"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", "my-ssh-key"),
				),
			},
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_private_key" "test" {
  name           = "updated-key"
  description    = "A test key"
  private_key    = "ssh-ed25519 AAAA-updated"
  is_git_related = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "pk-test-uuid-1"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", "updated-key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "description", "A test key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "private_key", "ssh-ed25519 AAAA-updated"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "is_git_related", "true"),
				),
			},
		},
	})
}

func TestPrivateKeyResource_Import(t *testing.T) {
	t.Parallel()
	srv := newPrivateKeyMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "import-key"
  private_key = "ssh-ed25519 AAAA-import"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "pk-test-uuid-1"),
				),
			},
			{
				ResourceName:                         "coolify_private_key.test",
				ImportState:                          true,
				ImportStateId:                        "pk-test-uuid-1",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestPrivateKeyResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newPrivateKeyMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "disappearing-key"
  private_key = "ssh-ed25519 AAAA-test-key"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_private_key.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_private_key.test", "/api/v1/security/keys/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
