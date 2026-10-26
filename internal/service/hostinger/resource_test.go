package hostinger_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestHostingerServerResource_CreateUpdateDelete(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var created bool
	var createBody map[string]any
	var patches []map[string]any
	description := ""
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/hostinger", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		created = true
		_ = json.Unmarshal(body, &createBody)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"aaaa0001-0001-4000-8000-000000000001","hostinger_virtual_machine_id":42,"ip":"203.0.113.10","provisioning":false}`))
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if deleted {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		desc := description
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"uuid":%q,"name":"web","description":%q,"ip":"203.0.113.10","port":22,"user":"root","private_key_uuid":"dddd0002-0002-4000-8000-000000000002","hostinger_virtual_machine_id":42,"hostinger_virtual_machine_status":"running","settings":{"concurrent_builds":2,"dynamic_timeout":3600,"deployment_queue_limit":25,"server_disk_usage_notification_threshold":80,"connection_timeout":10,"is_terminal_enabled":true}}`, r.PathValue("uuid"), desc)
	})
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var patch map[string]any
		_ = json.Unmarshal(body, &patch)
		mu.Lock()
		patches = append(patches, patch)
		if s, ok := patch["description"].(string); ok {
			description = s
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"aaaa0001-0001-4000-8000-000000000001"}`))
	})
	mux.HandleFunc("DELETE /api/v1/servers/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	base := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hostinger" "test" {
  name                       = "Web"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  item_id                    = "hostingercom-vps-kvm2"
  data_center_id             = 1
  template_id                = 7
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
  public_key_ids             = [3, 4]
  description                = "first"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: base,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_hostinger.test", "name", "Web"),
					resource.TestCheckResourceAttr("coolify_server_hostinger.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_server_hostinger.test", "hostinger_virtual_machine_id", "42"),
					resource.TestCheckResourceAttr("coolify_server_hostinger.test", "provisioning", "false"),
					resource.TestCheckResourceAttr("coolify_server_hostinger.test", "item_id", "hostingercom-vps-kvm2"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hostinger" "test" {
  name                       = "Web"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  item_id                    = "hostingercom-vps-kvm2"
  data_center_id             = 1
  template_id                = 7
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
  public_key_ids             = [3, 4]
  description                = "second"
}
`,
				Check: resource.TestCheckResourceAttr("coolify_server_hostinger.test", "description", "second"),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if !created {
		t.Fatal("create was not called")
	}
	if createBody["item_id"] != "hostingercom-vps-kvm2" || createBody["name"] != "Web" {
		t.Fatalf("create body = %#v", createBody)
	}
	if createBody["data_center_id"] != float64(1) || createBody["template_id"] != float64(7) {
		t.Fatalf("create placement = %#v", createBody)
	}
	if createBody["private_key_uuid"] != "dddd0002-0002-4000-8000-000000000002" {
		t.Fatalf("private_key_uuid = %#v", createBody["private_key_uuid"])
	}
	if createBody["enable_backups"] != false {
		t.Fatalf("enable_backups = %#v", createBody["enable_backups"])
	}
	ids, _ := createBody["public_key_ids"].([]any)
	if len(ids) != 2 {
		t.Fatalf("public_key_ids = %#v", createBody["public_key_ids"])
	}
	if len(patches) < 2 {
		t.Fatalf("patches = %d", len(patches))
	}
	if patches[0]["description"] != "first" {
		t.Fatalf("post-create patch = %#v", patches[0])
	}
	for i, patch := range patches {
		if _, ok := patch["item_id"]; ok {
			t.Fatalf("patch %d sent item_id: %#v", i, patch)
		}
	}
	if patches[len(patches)-1]["description"] != "second" {
		t.Fatalf("update patch = %#v", patches[len(patches)-1])
	}
	if !deleted {
		t.Fatal("delete was not called")
	}
}

func TestHostingerServerResource_Disappears(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/hostinger", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"aaaa0001-0001-4000-8000-000000000001","hostinger_virtual_machine_id":42,"ip":"203.0.113.10","provisioning":false}`))
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gone := deleted
		mu.Unlock()
		if gone {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"uuid":%q,"name":"web","ip":"203.0.113.10","port":22,"user":"root","private_key_uuid":"dddd0002-0002-4000-8000-000000000002","hostinger_virtual_machine_id":42,"settings":{"concurrent_builds":2,"dynamic_timeout":3600,"deployment_queue_limit":25,"server_disk_usage_notification_threshold":80,"connection_timeout":10,"is_terminal_enabled":true}}`, r.PathValue("uuid"))
	})
	mux.HandleFunc("DELETE /api/v1/servers/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hostinger" "test" {
  name                       = "web"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  item_id                    = "hostingercom-vps-kvm2"
  data_center_id             = 1
  template_id                = 7
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("coolify_server_hostinger.test", "uuid"),
				acctest.CheckResourceDisappears(srv.URL, "coolify_server_hostinger.test", "/api/v1/servers/"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestHostingerServerResource_OldCoolifySkipsPurchase(t *testing.T) {
	t.Parallel()
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/hostinger", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.2"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hostinger" "test" {
  name                       = "web"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  item_id                    = "hostingercom-vps-kvm2"
  data_center_id             = 1
  template_id                = 7
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}
`,
			ExpectError: regexp.MustCompile(`older than v4.5.0`),
		}},
	})
	if called {
		t.Fatal("older Coolify posted a purchase")
	}
}

func TestHostingerServerResource_ImportAdoptsPurchaseFields(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	posts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/hostinger", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		posts++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"aaaa0001-0001-4000-8000-000000000001","hostinger_virtual_machine_id":42,"ip":"203.0.113.10","provisioning":false}`))
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"uuid":%q,"name":"web","ip":"203.0.113.10","port":22,"user":"root","hostinger_virtual_machine_id":42,"hostinger_virtual_machine_status":"running","settings":{"concurrent_builds":2,"dynamic_timeout":3600,"deployment_queue_limit":25,"server_disk_usage_notification_threshold":80,"connection_timeout":10,"is_terminal_enabled":true}}`, r.PathValue("uuid"))
	})
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"aaaa0001-0001-4000-8000-000000000001"}`))
	})
	mux.HandleFunc("DELETE /api/v1/servers/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	cfg := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hostinger" "test" {
  name                       = "web"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  item_id                    = "hostingercom-vps-kvm2"
  data_center_id             = 1
  template_id                = 7
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}
`
	replaced := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hostinger" "test" {
  name                       = "web"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  item_id                    = "hostingercom-vps-kvm4"
  data_center_id             = 1
  template_id                = 7
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				ResourceName:       "coolify_server_hostinger.test",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      "aaaa0001-0001-4000-8000-000000000001",
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_server_hostinger.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("coolify_server_hostinger.test", "item_id", "hostingercom-vps-kvm2"),
			},
			{
				Config: replaced,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_server_hostinger.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if posts != 1 {
		t.Fatalf("purchases = %d, want 0 on import adopt and 1 on a known item_id change", posts)
	}
}
