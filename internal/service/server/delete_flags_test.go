package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDeleteServerResource_RemoveResourcesFromServer(t *testing.T) {
	t.Parallel()
	const uuid = "bbbb0001-0001-4000-8000-000000000099"

	t.Run("false on 4.7.6 fails before the request", func(t *testing.T) {
		t.Parallel()
		var hits atomic.Int32
		r := newDeleteFlagServer(t, "4.7.6", &hits, nil)
		resp := &resource.DeleteResponse{}
		r.deleteServerResource(context.Background(), serverResourceModel{
			UUID:                      types.StringValue(uuid),
			RemoveResourcesFromServer: types.BoolValue(false),
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected version error")
		}
		detail := resp.Diagnostics.Errors()[0].Detail()
		if !strings.Contains(detail, "Coolify >= v4.7.7") {
			t.Fatalf("detail = %q", detail)
		}
		if hits.Load() != 0 {
			t.Fatalf("DELETE hits = %d, want 0", hits.Load())
		}
	})

	t.Run("false on the nightly sends the query", func(t *testing.T) {
		t.Parallel()
		var query string
		r := newDeleteFlagServer(t, "4.8.0-alpha.0.20261121.1", nil, &query)
		resp := &resource.DeleteResponse{}
		r.deleteServerResource(context.Background(), serverResourceModel{
			UUID:                      types.StringValue(uuid),
			RemoveResourcesFromServer: types.BoolValue(false),
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("diagnostics: %v", resp.Diagnostics.Errors())
		}
		if query != "force=true&remove_resources_from_server=false" {
			t.Fatalf("query = %q", query)
		}
	})

	t.Run("default true is omitted", func(t *testing.T) {
		t.Parallel()
		var query string
		r := newDeleteFlagServer(t, "4.8.0", nil, &query)
		resp := &resource.DeleteResponse{}
		r.deleteServerResource(context.Background(), serverResourceModel{
			UUID:                      types.StringValue(uuid),
			RemoveResourcesFromServer: types.BoolValue(true),
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("diagnostics: %v", resp.Diagnostics.Errors())
		}
		if query != "force=true" {
			t.Fatalf("query = %q", query)
		}
	})

	t.Run("null is omitted", func(t *testing.T) {
		t.Parallel()
		var query string
		r := newDeleteFlagServer(t, "4.7.6", nil, &query)
		resp := &resource.DeleteResponse{}
		r.deleteServerResource(context.Background(), serverResourceModel{
			UUID:                      types.StringValue(uuid),
			RemoveResourcesFromServer: types.BoolNull(),
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("diagnostics: %v", resp.Diagnostics.Errors())
		}
		if query != "force=true" {
			t.Fatalf("query = %q", query)
		}
	})
}

func newDeleteFlagServer(t *testing.T, version string, hits *atomic.Int32, query *string) *serverResource {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			if hits != nil {
				hits.Add(1)
			}
			if query != nil {
				*query = r.URL.RawQuery
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	c := client.New(srv.URL, "test-token")
	c.CoolifyVersion = version
	return &serverResource{client: c}
}
