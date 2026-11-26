package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenService_DeleteBackupsPreservation(t *testing.T) {
	t.Parallel()
	svc := &client.Service{UUID: "svc-1", Name: "web"}

	t.Run("null becomes false", func(t *testing.T) {
		t.Parallel()
		model := serviceResourceModel{DeleteBackups: types.BoolNull()}
		flattenService(nil, svc, &model)
		if model.DeleteBackups.IsNull() || model.DeleteBackups.ValueBool() {
			t.Fatalf("DeleteBackups = %#v, want false", model.DeleteBackups)
		}
	})

	t.Run("true is preserved", func(t *testing.T) {
		t.Parallel()
		model := serviceResourceModel{DeleteBackups: types.BoolValue(true)}
		flattenService(nil, svc, &model)
		if !model.DeleteBackups.ValueBool() {
			t.Fatal("DeleteBackups overwritten")
		}
	})
}

func TestDeleteService_DeleteBackups(t *testing.T) {
	t.Parallel()
	const uuid = "svc-delete-backups"

	t.Run("true sends the query on 4.8.0", func(t *testing.T) {
		t.Parallel()
		var query string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				query = r.URL.RawQuery
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)
		c := client.New(srv.URL, "test-token")
		c.CoolifyVersion = "4.8.0"
		resp := &resource.DeleteResponse{}
		deleteService(context.Background(), c, "coolify_service", uuid, true, timeouts.Value{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("diagnostics: %v", resp.Diagnostics.Errors())
		}
		if query != "delete_backups=true" {
			t.Fatalf("query = %q", query)
		}
	})

	t.Run("true on 4.7.6 fails before the request", func(t *testing.T) {
		t.Parallel()
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				hits.Add(1)
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)
		c := client.New(srv.URL, "test-token")
		c.CoolifyVersion = "4.7.6"
		resp := &resource.DeleteResponse{}
		deleteService(context.Background(), c, "coolify_service", uuid, true, timeouts.Value{}, resp)
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
}
