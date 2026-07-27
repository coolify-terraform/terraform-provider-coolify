package destination_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func newDestinationMock(t *testing.T) *httptest.Server {
	t.Helper()
	store := map[string]*client.Destination{}
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/bbbb0001-0001-4000-8000-000000000001/destinations":
			var input client.CreateDestinationInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
				return
			}
			if input.Network == "" {
				t.Errorf("POST destination: expected non-empty network, got %+v", input)
				http.Error(w, `{"error":"network required"}`, http.StatusUnprocessableEntity)
				return
			}
			d := &client.Destination{
				UUID:       "dddd0001-0001-4000-8000-000000000001",
				Name:       input.Name,
				Network:    input.Network,
				Type:       "standalone",
				ServerUUID: "bbbb0001-0001-4000-8000-000000000001",
			}
			if input.Type != "" {
				d.Type = input.Type
			}
			if d.Name == "" {
				d.Name = "server-" + d.Network
			}
			store[d.UUID] = d
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(d)
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/api/v1/destinations/"):
			uuid := r.URL.Path[len("/api/v1/destinations/"):]
			d, ok := store[uuid]
			if !ok {
				http.Error(w, `{}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(d)
		case r.Method == http.MethodDelete:
			uuid := r.URL.Path[len("/api/v1/destinations/"):]
			delete(store, uuid)
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
}

func TestDestinationResource_Create(t *testing.T) {
	t.Parallel()
	srv := newDestinationMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_destination.test", "uuid", "dddd0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_destination.test", "network", "coolify-net"),
					resource.TestCheckResourceAttr("coolify_destination.test", "name", "my-net"),
					resource.TestCheckResourceAttr("coolify_destination.test", "type", "standalone"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDestinationResource_CreateSendsType(t *testing.T) {
	t.Parallel()
	var gotType string
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/bbbb0001-0001-4000-8000-000000000001/destinations" {
			var input client.CreateDestinationInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}
			if input.Network != "swarm-net" {
				t.Errorf("expected network swarm-net, got %q", input.Network)
			}
			gotType = input.Type
			d := &client.Destination{
				UUID:       "dddd0002-0002-4000-8000-000000000002",
				Name:       "swarm-dest",
				Network:    input.Network,
				Type:       input.Type,
				ServerUUID: "bbbb0001-0001-4000-8000-000000000001",
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(d)
			return
		}
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(&client.Destination{
				UUID:       "dddd0002-0002-4000-8000-000000000002",
				Name:       "swarm-dest",
				Network:    "swarm-net",
				Type:       "swarm",
				ServerUUID: "bbbb0001-0001-4000-8000-000000000001",
			})
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, `{}`, http.StatusNotFound)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "swarm-net"
  name        = "swarm-dest"
  type        = "swarm"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_destination.test", "type", "swarm"),
				),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if gotType != "swarm" {
		t.Fatalf("POST body type = %q, want swarm", gotType)
	}
}

func TestDestinationResource_Import(t *testing.T) {
	t.Parallel()
	srv := newDestinationMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_destination.test", "uuid", "dddd0001-0001-4000-8000-000000000001"),
				),
			},
			{
				ResourceName:                         "coolify_destination.test",
				ImportState:                          true,
				ImportStateId:                        "dddd0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestDestinationResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	srv := newDestinationMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
			},
			{
				ResourceName:  "coolify_destination.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestDestinationResource_Disappears(t *testing.T) {
	t.Parallel()
	srv := newDestinationMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_destination.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_destination.test", "/api/v1/destinations/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestDestinationResource_InvalidNetwork(t *testing.T) {
	t.Parallel()
	srv := newDestinationMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "-bad-start"
}`,
				ExpectError: regexp.MustCompile(`network must start with alphanumeric`),
			},
		},
	})
}

func TestDestinationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/bbbb0001-0001-4000-8000-000000000001/destinations", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "will-fail"
}`,
				ExpectError: regexp.MustCompile(`Error creating destination`),
			},
		},
	})
}
