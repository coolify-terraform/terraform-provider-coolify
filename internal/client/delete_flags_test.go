package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteBackupsOption(t *testing.T) {
	t.Parallel()
	calls := []struct {
		kind string
		path string
		call func(*Client, context.Context, string, DeleteBackupsOptions) error
	}{
		{"application", "/api/v1/applications/res-1", (*Client).DeleteApplicationWithOptions},
		{"database", "/api/v1/databases/res-1", (*Client).DeleteDatabaseWithOptions},
		{"service", "/api/v1/services/res-1", (*Client).DeleteServiceWithOptions},
	}
	cases := []struct {
		name    string
		version string
		send    bool
		query   string
		wantErr string
	}{
		{name: "false omits on 4.7.6", version: "4.7.6", query: ""},
		{name: "true on 4.8.0", version: "4.8.0", send: true, query: "delete_backups=true"},
		{name: "true on nightly alpha", version: "4.8.0-alpha.0.20261121.1", send: true, query: "delete_backups=true"},
		{name: "true on empty version", version: "", send: true, query: "delete_backups=true"},
		{name: "true on 4.3.0 lag string", version: "4.3.0", send: true, query: "delete_backups=true"},
		{name: "true on 4.7.6 is rejected", version: "4.7.6", send: true, wantErr: "Coolify >= v4.7.7"},
	}
	for _, kind := range calls {
		for _, tc := range cases {
			t.Run(kind.kind+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				var hits atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hits.Add(1)
					assert.Equal(t, http.MethodDelete, r.Method)
					assert.Equal(t, kind.path, r.URL.Path)
					assert.Equal(t, tc.query, r.URL.RawQuery)
					w.WriteHeader(http.StatusNoContent)
				}))
				t.Cleanup(srv.Close)

				c := New(srv.URL, "test-token")
				c.CoolifyVersion = tc.version
				err := kind.call(c, context.Background(), "res-1", DeleteBackupsOptions{DeleteBackups: tc.send})
				if tc.wantErr != "" {
					require.ErrorContains(t, err, tc.wantErr)
					require.ErrorContains(t, err, "this instance is 4.7.6")
					assert.Equal(t, int32(0), hits.Load())
					return
				}
				require.NoError(t, err)
				assert.Equal(t, int32(1), hits.Load())
			})
		}
	}
}

func TestDeleteServer_RemoveResourcesFromServer(t *testing.T) {
	t.Parallel()
	leave := false
	keep := true
	cases := []struct {
		name    string
		version string
		opts    DeleteServerOptions
		query   string
		wantErr string
	}{
		{name: "nil omits on 4.7.6", version: "4.7.6", query: "force=true"},
		{name: "true omits on 4.8.0", version: "4.8.0", opts: DeleteServerOptions{RemoveResourcesFromServer: &keep}, query: "force=true"},
		{name: "false on alpha", version: "4.8.0-alpha.0.20261121.1", opts: DeleteServerOptions{RemoveResourcesFromServer: &leave}, query: "force=true&remove_resources_from_server=false"},
		{name: "false on empty version", version: "", opts: DeleteServerOptions{RemoveResourcesFromServer: &leave}, query: "force=true&remove_resources_from_server=false"},
		{
			name:    "false with ssh key",
			version: "4.8.0",
			opts:    DeleteServerOptions{RemoveSSHKey: true, RemoveResourcesFromServer: &leave},
			query:   "force=true&remove_resources_from_server=false&remove_ssh_key=true",
		},
		{name: "false on 4.7.6 is rejected", version: "4.7.6", opts: DeleteServerOptions{RemoveResourcesFromServer: &leave}, wantErr: "Coolify >= v4.7.7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/api/v1/servers/srv-1", r.URL.Path)
				assert.Equal(t, tc.query, r.URL.RawQuery)
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(srv.Close)

			c := New(srv.URL, "test-token")
			c.CoolifyVersion = tc.version
			err := c.DeleteServerWithOptions(context.Background(), "srv-1", tc.opts)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.ErrorContains(t, err, "remove_resources_from_server")
				require.ErrorContains(t, err, "this instance is 4.7.6")
				assert.Equal(t, int32(0), hits.Load())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int32(1), hits.Load())
		})
	}
}
