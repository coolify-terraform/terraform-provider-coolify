package server_test

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

func newServerMockServer() *httptest.Server {
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			var input client.CreateServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			srv := &client.Server{
				UUID:           "bbbb0001-0001-4000-8000-000000000001",
				Name:           input.Name,
				Description:    input.Description,
				IP:             input.IP,
				Port:           input.Port,
				User:           input.User,
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsBuildServer:  input.IsBuildServer != nil && *input.IsBuildServer,
				IsReachable:    true,
				IsUsable:       true,
				Settings: &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
					IsTerminalEnabled:                    true,
				},
			}
			servers[srv.UUID] = srv
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(srv)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers":
			list := make([]client.Server, 0, len(servers))
			for _, srv := range servers {
				list = append(list, *srv)
			}
			json.NewEncoder(w).Encode(list)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			// Real Coolify GET omits private_key_uuid; return a copy
			// without it to exercise the flatten guard.
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
			if update.IsTerminalEnabled != nil {
				srv.Settings.IsTerminalEnabled = *update.IsTerminalEnabled
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

func TestServerResource_Create(t *testing.T) {
	t.Parallel()
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "my-server"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "uuid", "bbbb0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_server.test", "name", "my-server"),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "10.0.0.1"),
					resource.TestCheckResourceAttr("coolify_server.test", "port", "22"),
					resource.TestCheckResourceAttr("coolify_server.test", "user", "root"),
					resource.TestCheckResourceAttr("coolify_server.test", "private_key_uuid", "dddd0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_reachable", "true"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_usable", "true"),
					resource.TestCheckResourceAttr("coolify_server.test", "concurrent_builds", "2"),
					resource.TestCheckResourceAttr("coolify_server.test", "dynamic_timeout", "3600"),
					resource.TestCheckResourceAttr("coolify_server.test", "deployment_queue_limit", "25"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_disk_usage_notification_threshold", "80"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_disk_usage_check_frequency", "*/5 * * * *"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "my-server"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestServerResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()
	servers := make(map[string]*client.Server)
	var mu sync.Mutex
	var forceReadFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			var input client.CreateServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			created := &client.Server{UUID: "bbbb0009-0009-4000-8000-000000000009"}
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
resource "coolify_server" "test" {
  name             = "readback-failure"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Server created but refresh failed.*Could not read server.*partial Terraform state was saved`),
			},
		},
	})
}

func TestServerResource_Update(t *testing.T) {
	t.Parallel()
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "my-server"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "name", "my-server"),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "10.0.0.1"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + fmt.Sprintf(`
resource "coolify_server" "test" {
  name                                      = "updated-server"
  description                               = "Updated desc"
  ip                                        = "10.0.0.2"
  port                                      = %d
  user                                      = "deploy"
  private_key_uuid                          = "dddd0003-0003-4000-8000-000000000003"
  is_build_server                           = true
  concurrent_builds                         = 4
  dynamic_timeout                           = 7200
  deployment_queue_limit                    = 10
  connection_timeout                        = 30
  server_disk_usage_notification_threshold  = 90
  server_disk_usage_check_frequency         = "0 * * * *"
}`, 2222),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "uuid", "bbbb0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_server.test", "name", "updated-server"),
					resource.TestCheckResourceAttr("coolify_server.test", "description", "Updated desc"),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "10.0.0.2"),
					resource.TestCheckResourceAttr("coolify_server.test", "port", "2222"),
					resource.TestCheckResourceAttr("coolify_server.test", "user", "deploy"),
					resource.TestCheckResourceAttr("coolify_server.test", "private_key_uuid", "dddd0003-0003-4000-8000-000000000003"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "true"),
					resource.TestCheckResourceAttr("coolify_server.test", "concurrent_builds", "4"),
					resource.TestCheckResourceAttr("coolify_server.test", "dynamic_timeout", "7200"),
					resource.TestCheckResourceAttr("coolify_server.test", "deployment_queue_limit", "10"),
					resource.TestCheckResourceAttr("coolify_server.test", "connection_timeout", "30"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_disk_usage_notification_threshold", "90"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_disk_usage_check_frequency", "0 * * * *"),
				),
			},
		},
	})
}

func TestServerResource_Import(t *testing.T) {
	t.Parallel()
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "import-server"
  ip               = "10.0.0.5"
  private_key_uuid = "dddd0004-0004-4000-8000-000000000004"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "uuid", "bbbb0001-0001-4000-8000-000000000001"),
				),
			},
			{
				ResourceName:                         "coolify_server.test",
				ImportState:                          true,
				ImportStateId:                        "bbbb0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"private_key_uuid"},
			},
		},
	})
}

func TestServerResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "import-server"
  ip               = "10.0.0.5"
  private_key_uuid = "dddd0004-0004-4000-8000-000000000004"
}`,
			},
			{
				ResourceName:  "coolify_server.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestServerResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "disappearing-server"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_server.test", "/api/v1/servers/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestServerResource_DeleteUsesForce(t *testing.T) {
	t.Parallel()
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			var input client.CreateServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			created := &client.Server{
				UUID:           "bbbb0011-0001-4000-8000-000000000001",
				Name:           input.Name,
				IP:             input.IP,
				Port:           input.Port,
				User:           input.User,
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsBuildServer:  input.IsBuildServer != nil && *input.IsBuildServer,
				IsReachable:    true,
				IsUsable:       true,
				Settings: &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
					IsTerminalEnabled:                    true,
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
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "force-delete-server"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
			},
		},
	})
}

func TestServerResource_CreateWithSettings(t *testing.T) {
	t.Parallel()
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name              = "settings-server"
  ip                = "10.0.0.1"
  private_key_uuid  = "dddd0002-0002-4000-8000-000000000002"
  concurrent_builds = 8
  dynamic_timeout   = 1800
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "concurrent_builds", "8"),
					resource.TestCheckResourceAttr("coolify_server.test", "dynamic_timeout", "1800"),
					resource.TestCheckResourceAttr("coolify_server.test", "deployment_queue_limit", "25"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_disk_usage_notification_threshold", "80"),
				),
			},
		},
	})
}

func TestServerResource_UnsupportedExtendedSettingsAreRejectedBySchema(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "bad-extended-setting"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
  wildcard_domain  = "example.com"
}`,
				ExpectError: regexp.MustCompile(`Invalid Configuration for Read-Only Attribute|Cannot set value for this attribute as the provider has marked it as read-only`),
			},
		},
	})
}

func TestServerResource_InvalidPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "bad-port-server"
  ip               = "10.0.0.1"
  port             = 99999
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestServerResource_InvalidDeploymentQueueLimit(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name                   = "bad-queue-server"
  ip                     = "10.0.0.1"
  deployment_queue_limit = 0
  private_key_uuid       = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}

func TestServerResource_InvalidConnectionTimeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name               = "bad-timeout-server"
  ip                 = "10.0.0.1"
  connection_timeout = 301
  private_key_uuid   = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 300`),
			},
		},
	})
}

func TestServerResource_CreateAPIError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed: IP address already in use"}`, http.StatusUnprocessableEntity)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "will-fail"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`Error creating server`),
			},
		},
	})
}

func TestServerResource_CreateInstantValidate(t *testing.T) {
	t.Parallel()

	var gotCreate client.CreateServerInput
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			if err := json.NewDecoder(r.Body).Decode(&gotCreate); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			if gotCreate.InstantValidate == nil || !*gotCreate.InstantValidate {
				t.Errorf("POST instant_validate = %v, want true", gotCreate.InstantValidate)
			}
			created := &client.Server{
				UUID:           "bbbb0020-0001-4000-8000-000000000001",
				Name:           gotCreate.Name,
				IP:             gotCreate.IP,
				Port:           gotCreate.Port,
				User:           gotCreate.User,
				PrivateKeyUUID: gotCreate.PrivateKeyUUID,
				IsReachable:    true,
				IsUsable:       true,
				Settings: &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
					IsTerminalEnabled:                    true,
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
resource "coolify_server" "test" {
  name              = "validate-on-create"
  ip                = "10.0.0.1"
  private_key_uuid  = "dddd0002-0002-4000-8000-000000000002"
  instant_validate  = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "instant_validate", "true"),
				),
			},
		},
	})
}

func TestServerResource_UpdateInstantValidateAndTerminal(t *testing.T) {
	t.Parallel()

	var lastPatch client.UpdateServerInput
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			var input client.CreateServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			created := &client.Server{
				UUID:           "bbbb0021-0001-4000-8000-000000000001",
				Name:           input.Name,
				IP:             input.IP,
				Port:           input.Port,
				User:           input.User,
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
					IsTerminalEnabled:                    false,
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
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			server, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var update client.UpdateServerInput
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			lastPatch = update
			if update.Name != nil {
				server.Name = *update.Name
			}
			if update.IsTerminalEnabled != nil {
				server.Settings.IsTerminalEnabled = *update.IsTerminalEnabled
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
resource "coolify_server" "test" {
  name             = "terminal-off"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "is_terminal_enabled", "false"),
					resource.TestCheckResourceAttr("coolify_server.test", "instant_validate", "false"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name               = "terminal-on"
  ip                 = "10.0.0.1"
  private_key_uuid   = "dddd0002-0002-4000-8000-000000000002"
  instant_validate   = true
  is_terminal_enabled = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "is_terminal_enabled", "true"),
					resource.TestCheckResourceAttr("coolify_server.test", "instant_validate", "true"),
					func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if lastPatch.InstantValidate == nil || !*lastPatch.InstantValidate {
							return fmt.Errorf("PATCH instant_validate = %v, want true", lastPatch.InstantValidate)
						}
						if lastPatch.IsTerminalEnabled == nil || !*lastPatch.IsTerminalEnabled {
							return fmt.Errorf("PATCH is_terminal_enabled = %v, want true", lastPatch.IsTerminalEnabled)
						}
						return nil
					},
				),
			},
		},
	})
}

type server44Writes struct {
	mu     sync.Mutex
	create client.CreateServerInput
	patch  client.UpdateServerInput
}

func (w *server44Writes) setCreate(in client.CreateServerInput) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.create = in
}

func (w *server44Writes) setPatch(in client.UpdateServerInput) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.patch = in
}

func (w *server44Writes) snapshot() (client.CreateServerInput, client.UpdateServerInput) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.create, w.patch
}

func newCoolify44ServerMock(writes *server44Writes) *httptest.Server {
	servers := make(map[string]*client.Server)
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			var input client.CreateServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			writes.setCreate(input)
			if input.IsBuildServer != nil {
				http.Error(w, `{"message":"is_build_server is not allowed"}`, http.StatusUnprocessableEntity)
				return
			}
			role := "both"
			if input.ServerRole != nil && *input.ServerRole != "" {
				role = *input.ServerRole
			}
			created := &client.Server{
				UUID:          "bbbb0044-0001-4000-8000-000000000001",
				Name:          input.Name,
				IP:            input.IP,
				Port:          input.Port,
				User:          input.User,
				ServerRole:    role,
				IsReachable:   true,
				IsUsable:      true,
				IsBuildServer: role == "build",
				Settings: &client.ServerSettings{
					ServerRole:                           role,
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
					IsTerminalEnabled:                    true,
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
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			server, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var update client.UpdateServerInput
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			writes.setPatch(update)
			if update.IsBuildServer != nil {
				http.Error(w, `{"message":"is_build_server is not allowed"}`, http.StatusUnprocessableEntity)
				return
			}
			if update.ServerRole != nil {
				server.ServerRole = *update.ServerRole
				server.IsBuildServer = *update.ServerRole == "build"
				if server.Settings != nil {
					server.Settings.ServerRole = *update.ServerRole
				}
			}
			json.NewEncoder(w).Encode(server)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			delete(servers, strings.TrimPrefix(r.URL.Path, "/api/v1/servers/"))
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}), "4.4.0"))
}

func TestServerResource_DefaultCreateOmitsRoleOn44(t *testing.T) {
	t.Parallel()
	writes := &server44Writes{}
	srv := newCoolify44ServerMock(writes)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "default-role"
  ip               = "10.0.0.44"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "false"),
					func(*terraform.State) error {
						create, _ := writes.snapshot()
						if create.ServerRole != nil {
							return fmt.Errorf("POST server_role = %q, want omitted", *create.ServerRole)
						}
						if create.IsBuildServer != nil {
							return fmt.Errorf("POST is_build_server = %v, want omitted", *create.IsBuildServer)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestServerResource_ExplicitServerRoleOn44(t *testing.T) {
	t.Parallel()
	writes := &server44Writes{}
	srv := newCoolify44ServerMock(writes)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "role-deploy"
  ip               = "10.0.0.44"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
  server_role      = "deployment"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "server_role", "deployment"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "false"),
					func(*terraform.State) error {
						create, _ := writes.snapshot()
						if create.ServerRole == nil || *create.ServerRole != "deployment" {
							return fmt.Errorf("POST server_role = %v, want deployment", create.ServerRole)
						}
						if create.IsBuildServer != nil {
							return fmt.Errorf("POST is_build_server = %v, want omitted", *create.IsBuildServer)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestServerResource_ClearBuildServerOn44(t *testing.T) {
	t.Parallel()
	writes := &server44Writes{}
	srv := newCoolify44ServerMock(writes)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "build-then-both"
  ip               = "10.0.0.44"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
  is_build_server  = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "true"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_role", "build"),
					func(*terraform.State) error {
						create, _ := writes.snapshot()
						if create.ServerRole == nil || *create.ServerRole != "build" {
							return fmt.Errorf("POST server_role = %v, want build", create.ServerRole)
						}
						return nil
					},
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "build-then-both"
  ip               = "10.0.0.44"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
  is_build_server  = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("coolify_server.test", "server_role", "both"),
					func(*terraform.State) error {
						_, patch := writes.snapshot()
						if patch.ServerRole == nil || *patch.ServerRole != "both" {
							return fmt.Errorf("PATCH server_role = %v, want both", patch.ServerRole)
						}
						if patch.IsBuildServer != nil {
							return fmt.Errorf("PATCH is_build_server = %v, want omitted", *patch.IsBuildServer)
						}
						return nil
					},
				),
			},
		},
	})
}
