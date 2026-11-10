package s3storage_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestS3StorageResource_SFTPRejectsBucket(t *testing.T) {
	t.Parallel()
	var posts atomic.Int32
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"s3-1"}`))
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  bucket   = "b"
  key      = "user"
  secret   = "secret"
`),
			ExpectError: regexp.MustCompile(`bucket is only for s3 storage`),
		}},
	})
	if posts.Load() != 0 {
		t.Fatalf("sftp create with bucket sent %d POSTs", posts.Load())
	}
}

func TestS3StorageResource_OldCoolifyRejectsSFTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request %s %s on Coolify 4.6", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}), "4.6.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  key      = "user"
  secret   = "secret"
`),
			ExpectError: regexp.MustCompile(`Coolify version cannot set storage type`),
		}},
	})
}

func TestS3StorageResource_TypeChangeReplaces(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	stored := []byte(`{"uuid":"s3-1","name":"backups","endpoint":"https://s3.example.com","bucket":"b","region":"us-east-1","key":"access","type":"s3"}`)
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			req["uuid"] = "s3-1"
			raw, _ := json.Marshal(req)
			mu.Lock()
			stored = raw
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(raw)
		case http.MethodGet:
			mu.Lock()
			raw := append([]byte(nil), stored...)
			mu.Unlock()
			_, _ = w.Write(raw)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"message":"method"}`, http.StatusMethodNotAllowed)
		}
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "https://s3.example.com"
  bucket   = "b"
  region   = "us-east-1"
  key      = "access"
  secret   = "secret"
  type     = "s3"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  key      = "access"
  secret   = "secret"
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_s3_storage.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestS3StorageResource_SFTPUpdateKeepsPlaceholder(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patches atomic.Int32
	name := "backups"
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"s3-sftp"}`))
		case http.MethodGet:
			mu.Lock()
			body := fmt.Sprintf(`{"uuid":"s3-sftp","name":%q,"endpoint":"sftp://backup.example.com/backups","type":"sftp","region":"us-east-1","use_path_style_endpoint":true,"skip_tls_verification":false,"key":"user"}`, name)
			mu.Unlock()
			_, _ = w.Write([]byte(body))
		case http.MethodPatch:
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if _, ok := req["region"]; ok {
				t.Errorf("sftp update sent region: %v", req["region"])
			}
			if req["name"] != nil {
				mu.Lock()
				name, _ = req["name"].(string)
				mu.Unlock()
			}
			patches.Add(1)
			_, _ = w.Write([]byte(`{"uuid":"s3-sftp"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"message":"method"}`, http.StatusMethodNotAllowed)
		}
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  key      = "user"
  secret   = "secret"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups-renamed"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  key      = "user"
  secret   = "secret"
`),
			},
		},
	})
	if patches.Load() == 0 {
		t.Fatal("sftp rename did not PATCH")
	}
}

func TestS3StorageResource_SFTPKeyWithoutSecret(t *testing.T) {
	t.Parallel()
	var body map[string]any
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"s3-sftp"}`))
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"uuid":"s3-sftp","name":"backups","endpoint":"sftp://backup.example.com/backups","type":"sftp","region":"us-east-1","key":"user","private_key_uuid":"dddd0001-0001-4000-8000-000000000001"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name              = "backups"
  endpoint          = "sftp://backup.example.com/backups"
  type              = "sftp"
  key               = "user"
  private_key_uuid  = "dddd0001-0001-4000-8000-000000000001"
`),
			Check: resource.TestCheckResourceAttr("coolify_s3_storage.test", "private_key_uuid", "dddd0001-0001-4000-8000-000000000001"),
		}},
	})
	if _, ok := body["secret"]; ok {
		t.Fatalf("key login sent secret: %v", body["secret"])
	}
	if body["private_key_uuid"] != "dddd0001-0001-4000-8000-000000000001" {
		t.Fatalf("private_key_uuid = %v", body["private_key_uuid"])
	}
}

func TestS3StorageResource_SFTPClearPrivateKeySendsNull(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patched map[string]any
	key := "dddd0001-0001-4000-8000-000000000001"
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"s3-sftp"}`))
		case http.MethodGet:
			mu.Lock()
			id := key
			mu.Unlock()
			body := `{"uuid":"s3-sftp","name":"backups","endpoint":"sftp://backup.example.com/backups","type":"sftp","region":"us-east-1","key":"user"`
			if id != "" {
				body += `,"private_key_uuid":"` + id + `"`
			}
			_, _ = w.Write([]byte(body + `}`))
		case http.MethodPatch:
			raw, _ := io.ReadAll(r.Body)
			mu.Lock()
			_ = json.Unmarshal(raw, &patched)
			if _, ok := patched["private_key_uuid"]; ok && patched["private_key_uuid"] == nil {
				key = ""
			}
			mu.Unlock()
			_, _ = w.Write([]byte(`{"uuid":"s3-sftp"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name             = "backups"
  endpoint         = "sftp://backup.example.com/backups"
  type             = "sftp"
  key              = "user"
  private_key_uuid = "dddd0001-0001-4000-8000-000000000001"
`),
			},
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "sftp://backup.example.com/backups"
  type     = "sftp"
  key      = "user"
  secret   = "new-password"
`),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if patched["secret"] != "new-password" {
		t.Fatalf("secret = %v", patched["secret"])
	}
	if patched["private_key_uuid"] != nil {
		t.Fatalf("private_key_uuid = %#v, want JSON null", patched["private_key_uuid"])
	}
	if _, ok := patched["private_key_uuid"]; !ok {
		t.Fatal("private_key_uuid was omitted; Coolify keeps the key when the field is absent")
	}
}

func TestS3StorageResource_S3RequiresBucket(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/version" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}), "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name     = "backups"
  endpoint = "https://s3.example.com"
  key      = "access"
  secret   = "secret"
`),
			ExpectError: regexp.MustCompile(`bucket is required for s3 storage`),
		}},
	})
}
