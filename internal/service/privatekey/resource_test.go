package privatekey_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

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
				UUID:        "cccc0002-0002-4000-8000-000000000001",
				Name:        input.Name,
				Description: input.Description,
				PrivateKey:  input.PrivateKey,
				PublicKey:   "ssh-ed25519 AAAA-public",
				Fingerprint: "SHA256:test-fingerprint",
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_private_key", "/api/v1/security/keys/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "my-ssh-key"
  private_key = "ssh-ed25519 AAAA-test-key"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "cccc0002-0002-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", "my-ssh-key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "private_key", "ssh-ed25519 AAAA-test-key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "public_key", "ssh-ed25519 AAAA-public"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "fingerprint", "SHA256:test-fingerprint"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "is_git_related", "false"),
				),
			},
			// Plan idempotency
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
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

func TestPrivateKeyResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	keys := make(map[string]*client.PrivateKey)
	var mu sync.Mutex
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			key := &client.PrivateKey{UUID: "cccc0009-0009-4000-8000-000000000009"}
			keys[key.UUID] = key
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(key)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/"):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			key, ok := keys[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
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
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "readback-failure"
  private_key = "ssh-ed25519 AAAA-test-key"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Private key created but refresh failed.*Could not read private key.*partial Terraform state was saved`),
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
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "my-ssh-key"
  private_key = "ssh-ed25519 AAAA-test-key"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", "my-ssh-key"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "updated-key"
  description = "A test key"
  private_key = "ssh-ed25519 AAAA-updated"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "cccc0002-0002-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "name", "updated-key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "description", "A test key"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "private_key", "ssh-ed25519 AAAA-updated"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "public_key", "ssh-ed25519 AAAA-public"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "fingerprint", "SHA256:test-fingerprint"),
					resource.TestCheckResourceAttr("coolify_private_key.test", "is_git_related", "false"),
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
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "import-key"
  private_key = "ssh-ed25519 AAAA-import"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "cccc0002-0002-4000-8000-000000000001"),
				),
			},
			{
				ResourceName:                         "coolify_private_key.test",
				ImportState:                          true,
				ImportStateId:                        "cccc0002-0002-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestPrivateKeyResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := newPrivateKeyMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "import-key"
  private_key = "ssh-ed25519 AAAA-import"
}`,
			},
			{
				ResourceName:  "coolify_private_key.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

// TestPrivateKeyResource_DeleteRetry verifies that private key deletion
// retries on "in use" errors (Coolify's async app cleanup).
func TestPrivateKeyResource_DeleteRetry(t *testing.T) {
	t.Parallel()
	var deleteAttempts atomic.Int32
	keys := make(map[string]*client.PrivateKey)
	var mu sync.Mutex

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/security/keys":
			var input client.CreatePrivateKeyInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			key := &client.PrivateKey{
				UUID:        "retry-key-uuid-001",
				Name:        input.Name,
				PrivateKey:  input.PrivateKey,
				PublicKey:   "ssh-ed25519 AAAA-public",
				Fingerprint: "SHA256:test",
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

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/"):
			attempt := deleteAttempts.Add(1)
			if attempt <= 2 {
				http.Error(w, `{"message":"This key is in use and cannot be deleted."}`, http.StatusUnprocessableEntity)
				return
			}
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			delete(keys, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(s *terraform.State) error {
			if got := int(deleteAttempts.Load()); got < 3 {
				return fmt.Errorf("expected at least 3 delete attempts, got %d", got)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "retry-test-key"
  private_key = "ssh-ed25519 AAAA-retry"
}`,
				Check: resource.TestCheckResourceAttrSet("coolify_private_key.test", "uuid"),
			},
		},
	})
}

// TestPrivateKeyResource_PEMNormalization verifies that Read preserves the
// user's private key even when Coolify normalises it (trim + trailing newline).
func TestPrivateKeyResource_PEMNormalization(t *testing.T) {
	t.Parallel()
	keys := make(map[string]*client.PrivateKey)
	var mu sync.Mutex

	// Mock that mimics Coolify's formatPrivateKey(): trim + ensure trailing \n.
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			normalized := strings.TrimSpace(input.PrivateKey)
			if !strings.HasSuffix(normalized, "\n") {
				normalized += "\n"
			}
			key := &client.PrivateKey{
				UUID:        "cccc-norm-0001",
				Name:        input.Name,
				Description: input.Description,
				PrivateKey:  normalized,
				PublicKey:   "ssh-ed25519 AAAA-public",
				Fingerprint: "SHA256:test",
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

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/security/keys/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/security/keys/")
			delete(keys, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})))
	defer srv.Close()

	// User value deliberately has NO trailing newline.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "norm-key"
  private_key = "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_private_key.test", "uuid", "cccc-norm-0001"),
					// State should still hold the user's original (no trailing newline).
					resource.TestCheckResourceAttr("coolify_private_key.test", "private_key",
						"-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"),
				),
			},
			// Idempotency: plan after Read must be empty.
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "norm-key"
  private_key = "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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
				Config: acctest.ProviderBlockForURL(srv.URL) + `
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

func TestPrivateKeyResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/security/keys", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_private_key" "test" {
  name        = "will-fail"
  private_key = "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----"
}`,
				ExpectError: regexp.MustCompile(`Error creating private key`),
			},
		},
	})
}
