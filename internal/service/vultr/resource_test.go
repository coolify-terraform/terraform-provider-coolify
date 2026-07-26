package vultr_test

import (
	"encoding/json"
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
)

const (
	vultrTokenUUID  = "cccc0001-0001-4000-8000-000000000001"
	vultrKeyUUID    = "dddd0002-0002-4000-8000-000000000002"
	vultrServerUUID = "aaaa0001-0001-4000-8000-000000000001"
)

func defaultVultrServerSettings() *client.ServerSettings {
	return &client.ServerSettings{
		ConcurrentBuilds:                     2,
		DynamicTimeout:                       3600,
		DeploymentQueueLimit:                 25,
		ConnectionTimeout:                    10,
		ServerDiskUsageNotificationThreshold: 80,
		ServerDiskUsageCheckFrequency:        "*/5 * * * *",
	}
}

func applyVultrServerPatch(srv *client.Server, update client.UpdateServerInput) {
	if update.Name != nil {
		srv.Name = *update.Name
	}
	if update.Description != nil {
		srv.Description = *update.Description
	}
	if update.IP != nil {
		srv.IP = *update.IP
	}
	if update.Port != nil {
		srv.Port = *update.Port
	}
	if update.User != nil {
		srv.User = *update.User
	}
	if update.PrivateKeyUUID != nil {
		srv.PrivateKeyUUID = *update.PrivateKeyUUID
	}
	if update.IsBuildServer != nil {
		srv.IsBuildServer = *update.IsBuildServer
	}
	if srv.Settings == nil {
		srv.Settings = defaultVultrServerSettings()
	}
	if update.ConcurrentBuilds != nil {
		srv.Settings.ConcurrentBuilds = *update.ConcurrentBuilds
	}
	if update.DynamicTimeout != nil {
		srv.Settings.DynamicTimeout = *update.DynamicTimeout
	}
	if update.DeploymentQueueLimit != nil {
		srv.Settings.DeploymentQueueLimit = *update.DeploymentQueueLimit
	}
	if update.ConnectionTimeout != nil {
		srv.Settings.ConnectionTimeout = *update.ConnectionTimeout
	}
	if update.ServerDiskUsageNotificationThreshold != nil {
		srv.Settings.ServerDiskUsageNotificationThreshold = *update.ServerDiskUsageNotificationThreshold
	}
	if update.ServerDiskUsageCheckFrequency != nil {
		srv.Settings.ServerDiskUsageCheckFrequency = *update.ServerDiskUsageCheckFrequency
	}
}

func newVultrServerMock(t *testing.T) *httptest.Server {
	t.Helper()
	servers := map[string]*client.Server{}
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/vultr":
			var input client.CreateVultrServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			if input.Name == "" {
				t.Errorf("POST vultr: empty name")
			}
			if input.CloudProviderTokenUUID != vultrTokenUUID {
				t.Errorf("POST vultr: cloud_provider_token_uuid = %q, want %q", input.CloudProviderTokenUUID, vultrTokenUUID)
			}
			if input.Region != "ewr" {
				t.Errorf("POST vultr: region = %q, want ewr", input.Region)
			}
			if input.Plan != "vc2-1c-1gb" {
				t.Errorf("POST vultr: plan = %q, want vc2-1c-1gb", input.Plan)
			}
			if input.OsID != 1743 {
				t.Errorf("POST vultr: os_id = %d, want 1743", input.OsID)
			}
			if input.PrivateKeyUUID != vultrKeyUUID {
				t.Errorf("POST vultr: private_key_uuid = %q, want %q", input.PrivateKeyUUID, vultrKeyUUID)
			}
			if input.Name == "" || input.Region == "" || input.Plan == "" || input.OsID == 0 || input.PrivateKeyUUID == "" {
				http.Error(w, `{"error":"missing fields"}`, http.StatusUnprocessableEntity)
				return
			}
			srv := &client.Server{
				UUID:           vultrServerUUID,
				Name:           input.Name,
				IP:             "203.0.113.60",
				Port:           22,
				User:           "root",
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsReachable:    true,
				IsUsable:       true,
				Settings:       defaultVultrServerSettings(),
			}
			servers[srv.UUID] = srv
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(srv)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{}`, http.StatusNotFound)
				return
			}
			resp := *srv
			resp.PrivateKeyUUID = ""
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{}`, http.StatusNotFound)
				return
			}
			var update client.UpdateServerInput
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			applyVultrServerPatch(srv, update)
			_ = json.NewEncoder(w).Encode(srv)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			// CheckResourceDisappears and the provider both DELETE /servers/{uuid}
			// (provider adds ?force=true; disappears helper may not).
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			delete(servers, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
}

func vultrBaseConfig(name string) string {
	return `
resource "coolify_server_vultr" "test" {
  name                      = "` + name + `"
  cloud_provider_token_uuid = "` + vultrTokenUUID + `"
  region                    = "ewr"
  plan                      = "vc2-1c-1gb"
  os_id                     = 1743
  private_key_uuid          = "` + vultrKeyUUID + `"
}`
}

func TestVultrServerResource_Create(t *testing.T) {
	t.Parallel()
	srv := newVultrServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server_vultr", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("vu-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "uuid", vultrServerUUID),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "name", "vu-node"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "ip", "203.0.113.60"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "port", "22"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "user", "root"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "private_key_uuid", vultrKeyUUID),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "is_reachable", "true"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "is_usable", "true"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "concurrent_builds", "2"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "dynamic_timeout", "3600"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "connection_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "server_disk_usage_notification_threshold", "80"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "region", "ewr"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "plan", "vc2-1c-1gb"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "os_id", "1743"),
				),
			},
			{
				Config:             acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("vu-node"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestVultrServerResource_Update(t *testing.T) {
	t.Parallel()
	srv := newVultrServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("vu-node"),
				Check:  resource.TestCheckResourceAttr("coolify_server_vultr.test", "name", "vu-node"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_vultr" "test" {
  name                      = "vu-renamed"
  description               = "Updated description"
  cloud_provider_token_uuid = "` + vultrTokenUUID + `"
  region                    = "ewr"
  plan                      = "vc2-1c-1gb"
  os_id                     = 1743
  private_key_uuid          = "` + vultrKeyUUID + `"
  is_build_server           = true
  concurrent_builds         = 4
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "name", "vu-renamed"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "is_build_server", "true"),
					resource.TestCheckResourceAttr("coolify_server_vultr.test", "concurrent_builds", "4"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_vultr" "test" {
  name                      = "vu-renamed"
  description               = "Updated description"
  cloud_provider_token_uuid = "` + vultrTokenUUID + `"
  region                    = "ewr"
  plan                      = "vc2-1c-1gb"
  os_id                     = 1743
  private_key_uuid          = "` + vultrKeyUUID + `"
  is_build_server           = true
  concurrent_builds         = 4
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestVultrServerResource_Import(t *testing.T) {
	t.Parallel()
	srv := newVultrServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("vu-import"),
			},
			{
				ResourceName:                         "coolify_server_vultr.test",
				ImportState:                          true,
				ImportStateId:                        vultrServerUUID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid",
					"plan",
					"region",
					"os_id",
					"vultr_ssh_key_ids",
					"cloud_init_script",
					"instant_validate",
					"enable_ipv6",
					"disable_public_ipv4",
					"private_key_uuid",
				},
			},
		},
	})
}

func TestVultrServerResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := newVultrServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("vu-import"),
			},
			{
				ResourceName:  "coolify_server_vultr.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestVultrServerResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newVultrServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("vu-disappear"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server_vultr.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_server_vultr.test", "/api/v1/servers/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestVultrServerResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/vultr", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("will-fail"),
				ExpectError: regexp.MustCompile(`Error creating Vultr server`),
			},
		},
	})
}

func TestVultrServerResource_InvalidPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_vultr" "test" {
  name                      = "bad-port"
  cloud_provider_token_uuid = "` + vultrTokenUUID + `"
  region                    = "ewr"
  plan                      = "vc2-1c-1gb"
  os_id                     = 1743
  private_key_uuid          = "` + vultrKeyUUID + `"
  port                      = 99999
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestVultrServerResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	servers := map[string]*client.Server{}
	var mu sync.Mutex
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/vultr":
			created := &client.Server{UUID: "aaaa0009-0009-4000-8000-000000000009"}
			servers[created.UUID] = created
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(created)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			server, ok := servers[uuid]
			if !ok {
				http.Error(w, `{}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(server)
		case r.Method == http.MethodDelete:
			delete(servers, strings.TrimPrefix(r.URL.Path, "/api/v1/servers/"))
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      acctest.ProviderBlockForURL(srv.URL) + vultrBaseConfig("readback-failure"),
				ExpectError: regexp.MustCompile(`(?s)Vultr server created but refresh failed.*partial Terraform state was saved`),
			},
		},
	})
}
