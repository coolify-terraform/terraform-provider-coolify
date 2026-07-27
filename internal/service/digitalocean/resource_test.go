package digitalocean_test

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
	doTokenUUID  = "cccc0001-0001-4000-8000-000000000001"
	doKeyUUID    = "dddd0002-0002-4000-8000-000000000002"
	doServerUUID = "aaaa0001-0001-4000-8000-000000000001"
)

func defaultDOServerSettings() *client.ServerSettings {
	return &client.ServerSettings{
		ConcurrentBuilds:                     2,
		DynamicTimeout:                       3600,
		DeploymentQueueLimit:                 25,
		ConnectionTimeout:                    10,
		ServerDiskUsageNotificationThreshold: 80,
		ServerDiskUsageCheckFrequency:        "*/5 * * * *",
	}
}

func applyServerPatch(srv *client.Server, update client.UpdateServerInput) {
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
		srv.Settings = defaultDOServerSettings()
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

func newDOServerMock(t *testing.T) *httptest.Server {
	t.Helper()
	servers := map[string]*client.Server{}
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/digitalocean":
			var input client.CreateDigitalOceanServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			if input.Name == "" {
				t.Errorf("POST digitalocean: empty name")
			}
			if input.CloudProviderTokenUUID != doTokenUUID {
				t.Errorf("POST digitalocean: cloud_provider_token_uuid = %q, want %q", input.CloudProviderTokenUUID, doTokenUUID)
			}
			if input.Region != "nyc1" {
				t.Errorf("POST digitalocean: region = %q, want nyc1", input.Region)
			}
			if input.Size != "s-1vcpu-1gb" {
				t.Errorf("POST digitalocean: size = %q, want s-1vcpu-1gb", input.Size)
			}
			if input.Image != "ubuntu-24-04-x64" {
				t.Errorf("POST digitalocean: image = %v, want ubuntu-24-04-x64", input.Image)
			}
			if input.PrivateKeyUUID != doKeyUUID {
				t.Errorf("POST digitalocean: private_key_uuid = %q, want %q", input.PrivateKeyUUID, doKeyUUID)
			}
			if input.Name == "" || input.Region == "" || input.Size == "" || input.Image == nil || input.PrivateKeyUUID == "" {
				http.Error(w, `{"error":"missing fields"}`, http.StatusUnprocessableEntity)
				return
			}
			srv := &client.Server{
				UUID:           doServerUUID,
				Name:           input.Name,
				IP:             "203.0.113.50",
				Port:           22,
				User:           "root",
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsReachable:    true,
				IsUsable:       true,
				Settings:       defaultDOServerSettings(),
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
			applyServerPatch(srv, update)
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

func doBaseConfig(name string) string {
	return `
resource "coolify_server_digitalocean" "test" {
  name                      = "` + name + `"
  cloud_provider_token_uuid = "` + doTokenUUID + `"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "` + doKeyUUID + `"
}`
}

func TestDigitalOceanServerResource_Create(t *testing.T) {
	t.Parallel()
	srv := newDOServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server_digitalocean", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("do-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "uuid", doServerUUID),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "name", "do-node"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "ip", "203.0.113.50"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "port", "22"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "user", "root"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "private_key_uuid", doKeyUUID),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "is_reachable", "true"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "is_usable", "true"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "concurrent_builds", "2"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "dynamic_timeout", "3600"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "connection_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "server_disk_usage_notification_threshold", "80"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "region", "nyc1"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "size", "s-1vcpu-1gb"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "image", "ubuntu-24-04-x64"),
				),
			},
			{
				Config:             acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("do-node"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDigitalOceanServerResource_Update(t *testing.T) {
	t.Parallel()
	srv := newDOServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("do-node"),
				Check:  resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "name", "do-node"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_digitalocean" "test" {
  name                      = "do-renamed"
  description               = "Updated description"
  cloud_provider_token_uuid = "` + doTokenUUID + `"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "` + doKeyUUID + `"
  is_build_server           = true
  concurrent_builds         = 4
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "name", "do-renamed"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "is_build_server", "true"),
					resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "concurrent_builds", "4"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_digitalocean" "test" {
  name                      = "do-renamed"
  description               = "Updated description"
  cloud_provider_token_uuid = "` + doTokenUUID + `"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "` + doKeyUUID + `"
  is_build_server           = true
  concurrent_builds         = 4
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDigitalOceanServerResource_Import(t *testing.T) {
	t.Parallel()
	srv := newDOServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("do-import"),
			},
			{
				ResourceName:                         "coolify_server_digitalocean.test",
				ImportState:                          true,
				ImportStateId:                        doServerUUID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				// Create-only fields are not returned by the server GET endpoint.
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid",
					"size",
					"region",
					"image",
					"digitalocean_ssh_key_ids",
					"cloud_init_script",
					"instant_validate",
					"enable_ipv6",
					"monitoring",
					"private_key_uuid",
				},
			},
		},
	})
}

func TestDigitalOceanServerResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := newDOServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("do-import"),
			},
			{
				ResourceName:  "coolify_server_digitalocean.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestDigitalOceanServerResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newDOServerMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("do-disappear"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server_digitalocean.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_server_digitalocean.test", "/api/v1/servers/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestDigitalOceanServerResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/digitalocean", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("will-fail"),
				ExpectError: regexp.MustCompile(`Error creating DigitalOcean server`),
			},
		},
	})
}

func TestDigitalOceanServerResource_InvalidPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_digitalocean" "test" {
  name                      = "bad-port"
  cloud_provider_token_uuid = "` + doTokenUUID + `"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "` + doKeyUUID + `"
  port                      = 99999
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestDigitalOceanServerResource_InvalidSSHKeyIDs(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_digitalocean" "test" {
  name                      = "bad-ssh-keys"
  cloud_provider_token_uuid = "` + doTokenUUID + `"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "` + doKeyUUID + `"
  digitalocean_ssh_key_ids  = "10,not-an-id"
}`,
				ExpectError: regexp.MustCompile(`Invalid digitalocean_ssh_key_ids|comma-separated integer IDs`),
			},
		},
	})
}

func TestDigitalOceanServerResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	servers := map[string]*client.Server{}
	var mu sync.Mutex
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/digitalocean":
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
				Config:      acctest.ProviderBlockForURL(srv.URL) + doBaseConfig("readback-failure"),
				ExpectError: regexp.MustCompile(`(?s)DigitalOcean server created but refresh failed.*partial Terraform state was saved`),
			},
		},
	})
}
