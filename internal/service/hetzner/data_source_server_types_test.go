package hetzner_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestHetznerServerTypesDataSource(t *testing.T) {
	t.Parallel()

	serverTypes := []client.HetznerServerType{
		{ID: 1, Name: "cx11", Description: "CX11", Cores: 1, Memory: 2048, Disk: 20},
		{ID: 2, Name: "cx21", Description: "CX21", Cores: 2, Memory: 4096, Disk: 40},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hetzner/server-types", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(serverTypes)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_server_types" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.0.id", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.0.name", "cx11"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.0.cores", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.0.memory", "2048"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.0.disk", "20"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.1.name", "cx21"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_server_types.test", "server_types.1.cores", "2"),
				),
			},
		},
	})
}
