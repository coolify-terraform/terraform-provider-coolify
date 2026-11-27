package application

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

func TestFlattenApplicationCommon_DeleteBackupsPreservation(t *testing.T) {
	t.Parallel()
	app := &client.Application{UUID: "uuid-1", Name: "app"}

	t.Run("null becomes false", func(t *testing.T) {
		t.Parallel()
		f, _ := newDefaultFields()
		*f.DeleteBackups = types.BoolNull()
		flattenApplicationCommon(app, f)
		if f.DeleteBackups.ValueBool() {
			t.Fatal("DeleteBackups = true, want false for null")
		}
	})

	t.Run("unknown becomes false", func(t *testing.T) {
		t.Parallel()
		f, _ := newDefaultFields()
		*f.DeleteBackups = types.BoolUnknown()
		flattenApplicationCommon(app, f)
		if f.DeleteBackups.ValueBool() {
			t.Fatal("DeleteBackups = true, want false for unknown")
		}
	})

	t.Run("true is preserved", func(t *testing.T) {
		t.Parallel()
		f, _ := newDefaultFields()
		*f.DeleteBackups = types.BoolValue(true)
		flattenApplicationCommon(app, f)
		if !f.DeleteBackups.ValueBool() {
			t.Fatal("DeleteBackups = false, want true")
		}
	})
}

func TestDeleteApplication_DeleteBackups(t *testing.T) {
	t.Parallel()
	const uuid = "app-delete-backups"

	t.Run("true sends the query on 4.8.0", func(t *testing.T) {
		t.Parallel()
		var query string
		c := deleteBackupsClient(t, "4.8.0", &query)
		resp := &resource.DeleteResponse{}
		deleteApplication(context.Background(), c, "coolify_application", uuid, true, timeouts.Value{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete diagnostics: %v", resp.Diagnostics.Errors())
		}
		if query != "delete_backups=true" {
			t.Fatalf("query = %q", query)
		}
	})

	t.Run("true on 4.7.6 fails before the request", func(t *testing.T) {
		t.Parallel()
		var hits atomic.Int32
		c := deleteBackupsClientHits(t, "4.7.6", &hits)
		resp := &resource.DeleteResponse{}
		deleteApplication(context.Background(), c, "coolify_application", uuid, true, timeouts.Value{}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected version error")
		}
		detail := resp.Diagnostics.Errors()[0].Detail()
		if !strings.Contains(detail, "Coolify >= v4.7.7") || !strings.Contains(detail, uuid) {
			t.Fatalf("detail = %q", detail)
		}
		if hits.Load() != 0 {
			t.Fatalf("DELETE hits = %d, want 0", hits.Load())
		}
	})
}

func deleteBackupsClient(t *testing.T, version string, query *string) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			*query = r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	c := client.New(srv.URL, "test-token")
	c.CoolifyVersion = version
	return c
}

func deleteBackupsClientHits(t *testing.T, version string, hits *atomic.Int32) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			hits.Add(1)
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	c := client.New(srv.URL, "test-token")
	c.CoolifyVersion = version
	return c
}
