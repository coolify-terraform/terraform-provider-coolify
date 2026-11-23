package database

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

func TestFlattenDatabaseExtended_DeleteBackupsPreservation(t *testing.T) {
	t.Parallel()

	t.Run("null becomes false", func(t *testing.T) {
		t.Parallel()
		m := CommonModel{DeleteBackups: types.BoolNull()}
		FlattenDatabaseExtended(&client.Database{}, m.ExtFields())
		if m.DeleteBackups.IsNull() || m.DeleteBackups.ValueBool() {
			t.Fatalf("DeleteBackups = %#v, want false", m.DeleteBackups)
		}
	})

	t.Run("true is preserved", func(t *testing.T) {
		t.Parallel()
		m := CommonModel{DeleteBackups: types.BoolValue(true)}
		FlattenDatabaseExtended(&client.Database{}, m.ExtFields())
		if !m.DeleteBackups.ValueBool() {
			t.Fatal("DeleteBackups overwritten")
		}
	})
}

func TestDeleteDatabase_DeleteBackups(t *testing.T) {
	t.Parallel()
	const uuid = "db-delete-backups"

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
		err := DeleteDatabase(context.Background(), c, "coolify_database_postgresql", uuid, true, timeouts.Value{}, resp)
		if err != nil {
			t.Fatalf("delete: %v", err)
		}
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
		err := DeleteDatabase(context.Background(), c, "coolify_database_postgresql", uuid, true, timeouts.Value{}, resp)
		if err == nil || !strings.Contains(err.Error(), "Coolify >= v4.8.0") {
			t.Fatalf("err = %v", err)
		}
		if hits.Load() != 0 {
			t.Fatalf("DELETE hits = %d, want 0", hits.Load())
		}
	})
}
