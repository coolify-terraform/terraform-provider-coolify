package s3storage_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// mockS3Storage stores S3 storage data in the mock server.
type mockS3Storage struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Endpoint    string `json:"endpoint"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	Key         string `json:"key,omitempty"`
	Secret      string `json:"secret,omitempty"`
	IsUsable    bool   `json:"is_usable"`
}

func newMockCoolifyServer(auditT ...testing.TB) (*httptest.Server, *mockS3StorageStore) {
	return newMockCoolifyServerWithReadFailure(false, auditT...)
}

func newMockCoolifyServerWithReadFailure(forceReadFailure bool, auditT ...testing.TB) (*httptest.Server, *mockS3StorageStore) {
	store := &mockS3StorageStore{
		storages:        make(map[string]*mockS3Storage),
		omitSecretsRead: make(map[string]bool),
		counter:         0,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/s3-storages", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Endpoint    string `json:"endpoint"`
			Bucket      string `json:"bucket"`
			Region      string `json:"region"`
			Key         string `json:"key"`
			Secret      string `json:"secret"`
			IsUsable    *bool  `json:"is_usable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		if body.Name == "" || body.Endpoint == "" || body.Bucket == "" || body.Region == "" || body.Key == "" || body.Secret == "" {
			http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
			return
		}

		usable := false
		if body.IsUsable != nil {
			usable = *body.IsUsable
		}
		s := store.Create(body.Name, body.Description, body.Endpoint, body.Bucket, body.Region, body.Key, body.Secret, usable)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": s.UUID})
	})

	mux.HandleFunc("GET /api/v1/s3-storages/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if forceReadFailure {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		uuid := r.PathValue("uuid")
		s, ok := store.Get(uuid)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(s)
	})

	mux.HandleFunc("PATCH /api/v1/s3-storages/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		var body struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Endpoint    *string `json:"endpoint"`
			Bucket      *string `json:"bucket"`
			Region      *string `json:"region"`
			Key         *string `json:"key"`
			Secret      *string `json:"secret"`
			IsUsable    *bool   `json:"is_usable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		s, ok := store.Update(uuid, body.Name, body.Description, body.Endpoint, body.Bucket, body.Region, body.Key, body.Secret, body.IsUsable)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": s.UUID})
	})

	mux.HandleFunc("DELETE /api/v1/s3-storages/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if !store.Delete(uuid) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "S3 storage deleted."})
	})

	mux.HandleFunc("GET /api/v1/s3-storages", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(store.List())
	})

	mux.HandleFunc("POST /api/v1/s3-storages/{uuid}/validate", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if _, ok := store.Get(uuid); !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"valid": true})
	})

	handler := acctest.WithVersionEndpoint(mux)
	if len(auditT) > 0 {
		handler = spectest.WithSpecAudit(auditT[0], "coolify-v4", handler)
	}
	server := httptest.NewServer(handler)
	return server, store
}

type mockS3StorageStore struct {
	mu              sync.RWMutex
	storages        map[string]*mockS3Storage
	omitSecretsRead map[string]bool
	counter         int
}

func (s *mockS3StorageStore) Create(name, desc, endpoint, bucket, region, key, secret string, usable bool) *mockS3Storage {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	st := &mockS3Storage{
		UUID:        fmt.Sprintf("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380b%02d", s.counter),
		Name:        name,
		Description: desc,
		Endpoint:    endpoint,
		Bucket:      bucket,
		Region:      region,
		Key:         key,
		Secret:      secret,
		IsUsable:    usable,
	}
	s.storages[st.UUID] = st
	return st
}

func (s *mockS3StorageStore) Get(uuid string) (*mockS3Storage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.storages[uuid]
	if !ok {
		return nil, false
	}

	copy := *st
	if s.omitSecretsRead[uuid] {
		copy.Key = ""
		copy.Secret = ""
	}
	return &copy, true
}

func (s *mockS3StorageStore) Update(uuid string, name, desc, endpoint, bucket, region, key, secret *string, usable *bool) (*mockS3Storage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.storages[uuid]
	if !ok {
		return nil, false
	}
	if name != nil {
		st.Name = *name
	}
	if desc != nil {
		st.Description = *desc
	}
	if endpoint != nil {
		st.Endpoint = *endpoint
	}
	if bucket != nil {
		st.Bucket = *bucket
	}
	if region != nil {
		st.Region = *region
	}
	if key != nil {
		st.Key = *key
	}
	if secret != nil {
		st.Secret = *secret
	}
	if usable != nil {
		st.IsUsable = *usable
	}
	return st, true
}

func (s *mockS3StorageStore) OmitSecretsOnRead(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.storages[uuid]; !ok {
		return false
	}
	s.omitSecretsRead[uuid] = true
	return true
}

func (s *mockS3StorageStore) Delete(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.storages[uuid]
	if ok {
		delete(s.storages, uuid)
	}
	return ok
}

func (s *mockS3StorageStore) List() []*mockS3Storage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*mockS3Storage, 0, len(s.storages))
	for _, st := range s.storages {
		result = append(result, st)
	}
	return result
}

func testS3StorageConfig(url, name string) string {
	return acctest.ProviderBlockForURL(url) + fmt.Sprintf(`
resource "coolify_s3_storage" "test" {
  name        = %q
  description = "test storage"
  endpoint    = "https://s3.us-east-1.amazonaws.com"
  bucket      = "my-bucket"
  region      = "us-east-1"
  key         = "AKIATESTKEY"
  secret      = "secret-value"
}
`, name)
}

func TestS3StorageResource_Create(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer(t)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(server.URL, "coolify_s3_storage", "/api/v1/s3-storages/"),
		Steps: []resource.TestStep{
			{
				Config: testS3StorageConfig(server.URL, "my-s3-storage"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_s3_storage.test", "uuid"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", "my-s3-storage"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "description", "test storage"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "https://s3.us-east-1.amazonaws.com"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "my-bucket"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "us-east-1"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "key", "AKIATESTKEY"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "secret", "secret-value"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "is_usable", "false"),
				),
			},
			{
				Config:             testS3StorageConfig(server.URL, "my-s3-storage"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestS3StorageResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServerWithReadFailure(true)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config:      testS3StorageConfig(server.URL, "readback-failure"),
			ExpectError: regexp.MustCompile(`(?s)S3 storage created but refresh failed.*Could not read s3 storage.*partial Terraform state was saved`),
		}},
	})
}

func TestS3StorageResource_Update(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testS3StorageConfig(server.URL, "original-name"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", "original-name"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "my-bucket"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_s3_storage" "test" {
  name        = "updated-name"
  description = "updated description"
  endpoint    = "https://s3.eu-west-1.amazonaws.com"
  bucket      = "other-bucket"
  region      = "eu-west-1"
  key         = "NEWKEY"
  secret      = "new-secret"
  is_usable   = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "name", "updated-name"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "description", "updated description"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "endpoint", "https://s3.eu-west-1.amazonaws.com"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "bucket", "other-bucket"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "region", "eu-west-1"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "key", "NEWKEY"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "secret", "new-secret"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "is_usable", "true"),
				),
			},
		},
	})
}

func TestS3StorageResource_Import(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testS3StorageConfig(server.URL, "import-storage"),
			},
			{
				ResourceName:                         "coolify_s3_storage.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["coolify_s3_storage.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
		},
	})
}

func TestS3StorageResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	server, _ := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testS3StorageConfig(server.URL, "import-storage"),
			},
			{
				ResourceName:  "coolify_s3_storage.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestS3StorageResource_PreservesSecretsWhenReadOmitsThem(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testS3StorageConfig(server.URL, "preserve-secrets"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "key", "AKIATESTKEY"),
					resource.TestCheckResourceAttr("coolify_s3_storage.test", "secret", "secret-value"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_s3_storage.test"]
						if !ok {
							return fmt.Errorf("resource not found")
						}
						uuid := rs.Primary.Attributes["uuid"]
						if !store.OmitSecretsOnRead(uuid) {
							return fmt.Errorf("s3 storage %s not found in store", uuid)
						}
						return nil
					},
				),
			},
			{
				Config:             testS3StorageConfig(server.URL, "preserve-secrets"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestS3StorageResource_Disappears(t *testing.T) {
	t.Parallel()
	server, store := newMockCoolifyServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testS3StorageConfig(server.URL, "disappearing-storage"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_s3_storage.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_s3_storage.test"]
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

func TestS3StorageResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/s3-storages", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testS3StorageConfig(srv.URL, "will-fail"),
				ExpectError: regexp.MustCompile(`Error creating S3 storage`),
			},
		},
	})
}
