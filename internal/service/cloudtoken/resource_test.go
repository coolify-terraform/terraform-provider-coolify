package cloudtoken_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/SebTardif/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// mockCloudToken stores cloud token data in the mock server.
type mockCloudToken struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Token    string `json:"token,omitempty"`
}

// newMockCoolifyServer creates an httptest.Server that simulates the Coolify API for cloud tokens.
func newMockCoolifyServer(auditT ...testing.TB) (*httptest.Server, *mockCloudTokenStore) {
	store := &mockCloudTokenStore{
		cloudTokens: make(map[string]*mockCloudToken),
		counter:     0,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/cloud-tokens", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name     string `json:"name"`
			Provider string `json:"provider"`
			Token    string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		ct := store.Create(body.Name, body.Provider, body.Token)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": ct.UUID})
	})

	mux.HandleFunc("GET /api/v1/cloud-tokens/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		ct, ok := store.Get(uuid)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(ct)
	})

	mux.HandleFunc("PATCH /api/v1/cloud-tokens/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		var body struct {
			Name  *string `json:"name"`
			Token *string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		ct, ok := store.Update(uuid, body.Name, body.Token)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(ct)
	})

	mux.HandleFunc("DELETE /api/v1/cloud-tokens/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if !store.Delete(uuid) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("GET /api/v1/cloud-tokens", func(w http.ResponseWriter, r *http.Request) {
		tokens := store.List()
		json.NewEncoder(w).Encode(tokens)
	})

	mux.HandleFunc("POST /api/v1/cloud-tokens/{uuid}/validate", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		_, ok := store.Get(uuid)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "valid"})
	})

	var handler http.Handler = acctest.WithVersionEndpoint(mux)
	if len(auditT) > 0 {
		handler = spectest.WithSpecAudit(auditT[0], "coolify-v4", handler)
	}
	server := httptest.NewServer(handler)
	return server, store
}

// mockCloudTokenStore is a thread-safe in-memory store for mock cloud tokens.
type mockCloudTokenStore struct {
	mu          sync.RWMutex
	cloudTokens map[string]*mockCloudToken
	counter     int
}

func (s *mockCloudTokenStore) Create(name, prov, token string) *mockCloudToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	ct := &mockCloudToken{
		UUID:     fmt.Sprintf("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a%02d", s.counter),
		Name:     name,
		Provider: prov,
		Token:    token,
	}
	s.cloudTokens[ct.UUID] = ct
	return ct
}

func (s *mockCloudTokenStore) Get(uuid string) (*mockCloudToken, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ct, ok := s.cloudTokens[uuid]
	return ct, ok
}

func (s *mockCloudTokenStore) Update(uuid string, name, token *string) (*mockCloudToken, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ct, ok := s.cloudTokens[uuid]
	if !ok {
		return nil, false
	}
	if name != nil {
		ct.Name = *name
	}
	if token != nil {
		ct.Token = *token
	}
	return ct, true
}

func (s *mockCloudTokenStore) Delete(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.cloudTokens[uuid]
	if ok {
		delete(s.cloudTokens, uuid)
	}
	return ok
}

func (s *mockCloudTokenStore) List() []*mockCloudToken {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*mockCloudToken, 0, len(s.cloudTokens))
	for _, ct := range s.cloudTokens {
		result = append(result, ct)
	}
	return result
}

// testProviderFactory returns a provider factory wired to the given server URL.
func testProviderFactory(serverURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// providerConfig returns the HCL provider block pointing at the mock server.
func providerConfig(serverURL string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}
`, serverURL)
}

func TestCloudTokenResource_Create(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer(t)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		CheckDestroy:             acctest.CheckDestroy(server.URL, "coolify_cloud_token", "/api/v1/cloud-tokens/"),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "my-token"
  cloud_provider = "aws"
  token          = "secret-token-value"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_cloud_token.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "name", "my-token"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "cloud_provider", "aws"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "token", "secret-token-value"),
				),
			},
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "my-token"
  cloud_provider = "aws"
  token          = "secret-token-value"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestCloudTokenResource_Update(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "original-name"
  cloud_provider = "aws"
  token          = "original-token"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "name", "original-name"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "cloud_provider", "aws"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "token", "original-token"),
				),
			},
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "updated-name"
  cloud_provider = "aws"
  token          = "updated-token"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "name", "updated-name"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "cloud_provider", "aws"),
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "token", "updated-token"),
				),
			},
		},
	})
}

func TestCloudTokenResource_Import(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "import-token"
  cloud_provider = "hetzner"
  token          = "import-secret"
}
`,
			},
			{
				ResourceName:                         "coolify_cloud_token.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_cloud_token.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
		},
	})
}

func TestCloudTokenResource_Disappears(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactory(server.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "disappearing-token"
  cloud_provider = "aws"
  token          = "will-vanish"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_cloud_token.test", "uuid"),
					// Delete the cloud token out-of-band to simulate external deletion.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_cloud_token.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						store.Delete(uuid)
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
