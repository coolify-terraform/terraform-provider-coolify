package hetzner_test

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

func newHetznerServerMockServer() *httptest.Server {
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/hetzner":
			var input client.CreateHetznerServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			srv := &client.Server{
				UUID:           "aaaa0001-0001-4000-8000-000000000001",
				Name:           input.Name,
				IP:             "203.0.113.42",
				Port:           22,
				User:           "root",
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsReachable:    true,
				IsUsable:       true,
				Settings: &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
				},
			}
			servers[srv.UUID] = srv
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(srv)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			resp := *srv
			resp.PrivateKeyUUID = ""
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var update client.UpdateServerInput
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
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
				srv.Settings = &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
				}
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
			json.NewEncoder(w).Encode(srv)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			delete(servers, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})))
}

func TestHetznerServerResource_Create(t *testing.T) {
	t.Parallel()
	srv := newHetznerServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server_hetzner", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "my-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "name", "my-hetzner"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "ip", "203.0.113.42"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "port", "22"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "user", "root"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "private_key_uuid", "dddd0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "is_reachable", "true"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "is_usable", "true"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "concurrent_builds", "2"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "dynamic_timeout", "3600"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "connection_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "server_disk_usage_notification_threshold", "80"),
				),
			},
			// Idempotent plan check.
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "my-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestHetznerServerResource_CreateWithSettings(t *testing.T) {
	t.Parallel()
	srv := newHetznerServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                                   = "my-hetzner"
  description                            = "Build node"
  cloud_provider_token_uuid              = "cccc0001-0001-4000-8000-000000000001"
  server_type                            = "cx22"
  location                               = "fsn1"
  image                                  = "ubuntu-24.04"
  port                                   = 2222
  user                                   = "deployer"
  private_key_uuid                       = "dddd0002-0002-4000-8000-000000000002"
  is_build_server                        = true
  concurrent_builds                      = 8
  dynamic_timeout                        = 1800
  deployment_queue_limit                 = 10
  server_disk_usage_notification_threshold = 90
  server_disk_usage_check_frequency      = "0 * * * *"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "description", "Build node"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "port", "2222"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "user", "deployer"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "is_build_server", "true"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "concurrent_builds", "8"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "dynamic_timeout", "1800"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "deployment_queue_limit", "10"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "connection_timeout", "10"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "server_disk_usage_notification_threshold", "90"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "server_disk_usage_check_frequency", "0 * * * *"),
				),
			},
			// Verify single-apply convergence: no second apply needed.
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                                   = "my-hetzner"
  description                            = "Build node"
  cloud_provider_token_uuid              = "cccc0001-0001-4000-8000-000000000001"
  server_type                            = "cx22"
  location                               = "fsn1"
  image                                  = "ubuntu-24.04"
  port                                   = 2222
  user                                   = "deployer"
  private_key_uuid                       = "dddd0002-0002-4000-8000-000000000002"
  is_build_server                        = true
  concurrent_builds                      = 8
  dynamic_timeout                        = 1800
  deployment_queue_limit                 = 10
  server_disk_usage_notification_threshold = 90
  server_disk_usage_check_frequency      = "0 * * * *"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestHetznerServerResource_Update(t *testing.T) {
	t.Parallel()
	srv := newHetznerServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "my-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_hetzner.test", "name", "my-hetzner"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "renamed-hetzner"
  description                = "Updated description"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
  is_build_server            = true
  concurrent_builds          = 4
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "name", "renamed-hetzner"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "is_build_server", "true"),
					resource.TestCheckResourceAttr("coolify_server_hetzner.test", "concurrent_builds", "4"),
				),
			},
			// Post-update idempotency check.
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "renamed-hetzner"
  description                = "Updated description"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
  is_build_server            = true
  concurrent_builds          = 4
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestHetznerServerResource_Import(t *testing.T) {
	t.Parallel()
	srv := newHetznerServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "my-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
			},
			{
				ResourceName:                         "coolify_server_hetzner.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				// Hetzner create-only fields are not returned by the server GET endpoint.
				ImportStateVerifyIgnore: []string{
					"cloud_provider_token_uuid",
					"server_type",
					"location",
					"image",
					"hetzner_ssh_key_ids",
					"cloud_init_script",
					"instant_validate",
					"enable_ipv4",
					"enable_ipv6",
					"private_key_uuid",
				},
			},
		},
	})
}

func TestHetznerServerResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := newHetznerServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "my-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
			},
			{
				ResourceName:  "coolify_server_hetzner.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestHetznerServerResource_InvalidDeploymentQueueLimit(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "bad-queue-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
  deployment_queue_limit     = 0
}`,
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}

func TestHetznerServerResource_InvalidConnectionTimeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "bad-timeout-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
  connection_timeout         = 301
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 300`),
			},
		},
	})
}

func TestHetznerServerResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	servers := make(map[string]*client.Server)
	var mu sync.Mutex
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/hetzner":
			created := &client.Server{UUID: "aaaa0009-0009-4000-8000-000000000009"}
			servers[created.UUID] = created
			forceReadFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(created)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			if forceReadFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			server, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(server)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			delete(servers, uuid)
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
resource "coolify_server_hetzner" "test" {
  name                       = "readback-failure"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`(?s)Hetzner server created but refresh failed.*partial Terraform state was saved`),
			},
		},
	})
}

func TestHetznerServerResource_InvalidPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "bad-port-server"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  port                       = 99999
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestHetznerServerResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newHetznerServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "my-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server_hetzner.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_server_hetzner.test", "/api/v1/servers/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestHetznerServerResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/hetzner", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                      = "will-fail"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  server_type               = "cx22"
  location                  = "fsn1"
  image                     = "ubuntu-24.04"
  private_key_uuid          = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`Error creating Hetzner server`),
			},
		},
	})
}

func TestHetznerServerResource_DeleteUsesForce(t *testing.T) {
	t.Parallel()
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/hetzner":
			var input client.CreateHetznerServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			created := &client.Server{
				UUID:           "aaaa0011-0001-4000-8000-000000000001",
				Name:           input.Name,
				IP:             "203.0.113.42",
				Port:           22,
				User:           "root",
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsReachable:    true,
				IsUsable:       true,
				Settings: &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
				},
			}
			servers[created.UUID] = created
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(created)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			server, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			resp := *server
			resp.PrivateKeyUUID = ""
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			if r.URL.Query().Get("force") != "true" {
				http.Error(w, `{"message":"Server has resources. Use ?force=true to delete all resources and the server, or delete resources manually first."}`, http.StatusBadRequest)
				return
			}
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			delete(servers, uuid)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Server deleted."})

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server_hetzner", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_hetzner" "test" {
  name                       = "force-delete-hetzner"
  cloud_provider_token_uuid  = "cccc0001-0001-4000-8000-000000000001"
  server_type                = "cx22"
  location                   = "fsn1"
  image                      = "ubuntu-24.04"
  private_key_uuid           = "dddd0002-0002-4000-8000-000000000002"
}`,
			},
		},
	})
}
