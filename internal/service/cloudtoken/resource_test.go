package cloudtoken_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/spectest"
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
	return newMockCoolifyServerWithReadFailure(false, auditT...)
}

func newMockCoolifyServerWithReadFailure(forceReadFailure bool, auditT ...testing.TB) (*httptest.Server, *mockCloudTokenStore) {
	store := &mockCloudTokenStore{
		cloudTokens:     make(map[string]*mockCloudToken),
		omitTokenOnRead: make(map[string]bool),
		counter:         0,
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
		if forceReadFailure {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
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

	handler := acctest.WithVersionEndpoint(mux)
	if len(auditT) > 0 {
		handler = spectest.WithSpecAudit(auditT[0], "coolify-v4", handler)
	}
	server := httptest.NewServer(handler)
	return server, store
}

// mockCloudTokenStore is a thread-safe in-memory store for mock cloud tokens.
type mockCloudTokenStore struct {
	mu              sync.RWMutex
	cloudTokens     map[string]*mockCloudToken
	omitTokenOnRead map[string]bool
	counter         int
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
	if !ok {
		return nil, false
	}

	copy := *ct
	if s.omitTokenOnRead[uuid] {
		copy.Token = ""
	}

	return &copy, true
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

func (s *mockCloudTokenStore) OmitTokenOnRead(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cloudTokens[uuid]; !ok {
		return false
	}
	s.omitTokenOnRead[uuid] = true
	return true
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

func TestCloudTokenResource_Create(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer(t)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(server.URL, "coolify_cloud_token", "/api/v1/cloud-tokens/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

func TestCloudTokenResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServerWithReadFailure(true)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "readback-failure"
  cloud_provider = "aws"
  token          = "secret-token-value"
}
`,
			ExpectError: regexp.MustCompile(`(?s)Cloud token created but refresh failed.*Could not read cloud token.*partial Terraform state was saved`),
		}},
	})
}

func TestCloudTokenResource_Update(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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

func TestCloudTokenResource_PreservesTokenWhenReadOmitsIt(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "preserve-token"
  cloud_provider = "aws"
  token          = "keep-me"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_cloud_token.test", "token", "keep-me"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_cloud_token.test"]
						if !ok {
							return fmt.Errorf("resource not found")
						}

						uuid := rs.Primary.Attributes["uuid"]
						if !store.OmitTokenOnRead(uuid) {
							return fmt.Errorf("cloud token %s not found in store", uuid)
						}
						return nil
					},
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_cloud_token" "test" {
  name           = "preserve-token"
  cloud_provider = "aws"
  token          = "keep-me"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestCloudTokenResource_Disappears(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
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
