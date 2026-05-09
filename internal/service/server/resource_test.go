package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

func testProviderBlock(serverURL string) string {
	return `
provider "coolify" {
  endpoint  = "` + serverURL + `"
  token = "test-token"
}`
}

func newServerMockServer() *httptest.Server {
	servers := make(map[string]*client.Server)
	var mu sync.Mutex

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				UUID:           "srv-test-uuid-1",
				Name:           input.Name,
				Description:    input.Description,
				IP:             input.IP,
				Port:           input.Port,
				User:           input.User,
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsBuildServer:  input.IsBuildServer,
				IsReachable:    true,
				IsUsable:       true,
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
			json.NewEncoder(w).Encode(srv)

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
			json.NewEncoder(w).Encode(srv)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			delete(servers, uuid)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
}

func TestServerResource_Create(t *testing.T) {
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_server" "test" {
  name             = "my-server"
  ip               = "10.0.0.1"
  private_key_uuid = "pk-uuid-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "uuid", "srv-test-uuid-1"),
					resource.TestCheckResourceAttr("coolify_server.test", "name", "my-server"),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "10.0.0.1"),
					resource.TestCheckResourceAttr("coolify_server.test", "port", "22"),
					resource.TestCheckResourceAttr("coolify_server.test", "user", "root"),
					resource.TestCheckResourceAttr("coolify_server.test", "private_key_uuid", "pk-uuid-1"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_reachable", "true"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_usable", "true"),
				),
			},
		},
	})
}

func TestServerResource_Update(t *testing.T) {
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_server" "test" {
  name             = "my-server"
  ip               = "10.0.0.1"
  private_key_uuid = "pk-uuid-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "name", "my-server"),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "10.0.0.1"),
				),
			},
			{
				Config: testProviderBlock(srv.URL) + fmt.Sprintf(`
resource "coolify_server" "test" {
  name             = "updated-server"
  description      = "Updated desc"
  ip               = "10.0.0.2"
  port             = %d
  user             = "deploy"
  private_key_uuid = "pk-uuid-2"
  is_build_server  = true
}`, 2222),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "uuid", "srv-test-uuid-1"),
					resource.TestCheckResourceAttr("coolify_server.test", "name", "updated-server"),
					resource.TestCheckResourceAttr("coolify_server.test", "description", "Updated desc"),
					resource.TestCheckResourceAttr("coolify_server.test", "ip", "10.0.0.2"),
					resource.TestCheckResourceAttr("coolify_server.test", "port", "2222"),
					resource.TestCheckResourceAttr("coolify_server.test", "user", "deploy"),
					resource.TestCheckResourceAttr("coolify_server.test", "private_key_uuid", "pk-uuid-2"),
					resource.TestCheckResourceAttr("coolify_server.test", "is_build_server", "true"),
				),
			},
		},
	})
}

func TestServerResource_Import(t *testing.T) {
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_server" "test" {
  name             = "import-server"
  ip               = "10.0.0.5"
  private_key_uuid = "pk-uuid-5"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server.test", "uuid", "srv-test-uuid-1"),
				),
			},
			{
				ResourceName:                         "coolify_server.test",
				ImportState:                          true,
				ImportStateId:                        "srv-test-uuid-1",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestServerResource_Disappears(t *testing.T) {
	srv := newServerMockServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(srv.URL) + `
resource "coolify_server" "test" {
  name             = "disappearing-server"
  ip               = "10.0.0.1"
  private_key_uuid = "pk-uuid-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_server.test", "uuid"),
					// Delete the server out-of-band via the mock API.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_server.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/servers/"+uuid, nil)
						if err != nil {
							return err
						}
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
