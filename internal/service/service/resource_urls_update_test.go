package service_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type serviceURLFixture struct {
	mu          sync.Mutex
	description string
	apps        []map[string]string
	patches     []map[string]any
	deleted     bool
}

func (f *serviceURLFixture) handler(uuid string) http.Handler {
	return acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			f.recordWrite(body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services/"+uuid:
			if f.deleted {
				http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
				return
			}
			payload := map[string]any{
				"uuid":             uuid,
				"name":             "urls-svc",
				"type":             "plausible",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"applications":     f.apps,
			}
			if f.description != "" {
				payload["description"] = f.description
			}
			_ = json.NewEncoder(w).Encode(payload)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/services/"+uuid:
			var decoded map[string]any
			_ = json.Unmarshal(body, &decoded)
			f.patches = append(f.patches, decoded)
			f.recordWrite(body)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/services/"+uuid:
			f.deleted = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
}

func (f *serviceURLFixture) recordWrite(body []byte) {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return
	}
	if desc, ok := decoded["description"].(string); ok {
		f.description = desc
	}
	urls, ok := decoded["urls"].([]any)
	if !ok {
		return
	}
	apps := make([]map[string]string, 0, len(urls))
	for _, raw := range urls {
		entry, _ := raw.(map[string]any)
		name, _ := entry["name"].(string)
		url, _ := entry["url"].(string)
		if url == "" {
			continue
		}
		apps = append(apps, map[string]string{"name": name, "fqdn": url})
	}
	f.apps = apps
}

func TestServiceResource_ClearURLs(t *testing.T) {
	t.Parallel()
	const svcUUID = "svc-clear-urls-001"
	fix := &serviceURLFixture{}
	srv := httptest.NewServer(fix.handler(svcUUID))
	defer srv.Close()

	base := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: base + `
  urls = [{ name = "web", url = "https://web.example.com" }]
}
`,
				Check: resource.TestCheckResourceAttr("coolify_service.test", "urls.0.name", "web"),
			},
			{
				Config: base + `
  description = "renamed"
  urls = [{ name = "web", url = "https://web.example.com" }]
}
`,
			},
			{
				Config: base + `
  description = "renamed"
}
`,
				Check: resource.TestCheckNoResourceAttr("coolify_service.test", "urls.0.name"),
			},
		},
	})

	fix.mu.Lock()
	defer fix.mu.Unlock()
	if len(fix.patches) < 2 {
		t.Fatalf("patches = %d, want description update and clear", len(fix.patches))
	}
	if _, ok := fix.patches[0]["urls"]; ok {
		t.Fatalf("description update sent urls: %#v", fix.patches[0]["urls"])
	}
	if _, ok := fix.patches[0]["force_domain_override"]; ok {
		t.Fatalf("description update sent force_domain_override: %#v", fix.patches[0]["force_domain_override"])
	}
	urls, ok := fix.patches[1]["urls"].([]any)
	if !ok || len(urls) != 1 {
		t.Fatalf("clear urls = %#v", fix.patches[1]["urls"])
	}
	entry, _ := urls[0].(map[string]any)
	if entry["name"] != "web" || entry["url"] != "" {
		t.Fatalf("clear entry = %#v, want web with empty url", entry)
	}
}

func TestServiceResource_ForceDomainFromConfigUnderIgnoreChanges(t *testing.T) {
	t.Parallel()
	const svcUUID = "svc-force-urls-001"
	fix := &serviceURLFixture{}
	srv := httptest.NewServer(fix.handler(svcUUID))
	defer srv.Close()

	provider := acctest.ProviderBlockForURL(srv.URL)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: provider + `
resource "coolify_service" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  type         = "plausible"
  urls = [{ name = "web", url = "https://old.example.com" }]
}
`,
			},
			{
				Config: provider + `
resource "coolify_service" "test" {
  project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
  type                    = "plausible"
  description             = "shared domain"
  force_domain_override   = true
  urls = [{ name = "web", url = "https://new.example.com" }]
  lifecycle {
    ignore_changes = [force_domain_override]
  }
}
`,
			},
		},
	})

	fix.mu.Lock()
	defer fix.mu.Unlock()
	if len(fix.patches) != 1 {
		t.Fatalf("patches = %d, want the url change", len(fix.patches))
	}
	if fix.patches[0]["force_domain_override"] != true {
		t.Fatalf("force_domain_override = %#v, want true from configuration", fix.patches[0]["force_domain_override"])
	}
	urls, _ := fix.patches[0]["urls"].([]any)
	if len(urls) != 1 {
		t.Fatalf("urls = %#v", fix.patches[0]["urls"])
	}
}
