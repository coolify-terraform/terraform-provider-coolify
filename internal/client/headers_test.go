package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ExtraHeaders_SentOnEveryRequest(t *testing.T) {
	t.Parallel()
	var (
		mu   sync.Mutex
		seen []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer proxy-jwt", r.Header.Get("Proxy-Authorization"))
		assert.Equal(t, "tenant-a", r.Header.Get("X-Tenant"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode("4.4.0")
		case "/api/v1/projects":
			if r.Method == http.MethodPost {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"uuid":"p1"}`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token", RetryConfig{Headers: map[string]string{
		"proxy-authorization": "Bearer proxy-jwt",
		"X-Tenant":            "tenant-a",
	}})
	ctx := context.Background()

	_, err := c.GetVersion(ctx) // doText
	require.NoError(t, err)
	var projects []Project
	err = c.doCachedList(ctx, "/api/v1/projects", &projects) // doCachedList
	require.NoError(t, err)
	_, err = c.CreateProject(ctx, CreateProjectInput{Name: "p"}) // doWithStatus
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"GET /api/v1/version", "GET /api/v1/projects", "POST /api/v1/projects"}, seen)
}

func TestClient_ExtraHeaders_CannotReplaceProviderHeaders(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"Bearer test-token"}, r.Header.Values("Authorization"))
		assert.Equal(t, []string{"terraform-provider-coolify"}, r.Header.Values("User-Agent"))
		_ = json.NewEncoder(w).Encode("4.4.0")
	}))
	defer srv.Close()

	// New does not validate; setCommonHeaders still sets the provider's
	// own headers last, so they win over an unvalidated user header.
	c := New(srv.URL, "test-token", RetryConfig{Headers: map[string]string{
		"Authorization": "Bearer someone-else",
		"User-Agent":    "other",
	}})
	_, err := c.GetVersion(context.Background())
	require.NoError(t, err)
}

func TestClient_ExtraHeaders_EmptyChangesNothing(t *testing.T) {
	t.Parallel()
	for name, headers := range map[string]map[string]string{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := make([]string, 0, len(r.Header))
				for k := range r.Header {
					got = append(got, k)
				}
				assert.ElementsMatch(t, []string{"Authorization", "User-Agent", "Accept-Encoding"}, got)
				_ = json.NewEncoder(w).Encode("4.4.0")
			}))
			defer srv.Close()

			c := New(srv.URL, "test-token", RetryConfig{Headers: headers})
			assert.Nil(t, c.extraHeaders)
			_, err := c.GetVersion(context.Background())
			require.NoError(t, err)
		})
	}
}

func TestRetryConfig_ValidateHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     RetryConfig
		wantErr []string
	}{
		{name: "nil", cfg: RetryConfig{}},
		{name: "empty", cfg: RetryConfig{Headers: map[string]string{}}},
		{
			name: "proxy headers",
			cfg: RetryConfig{Headers: map[string]string{
				"Proxy-Authorization": "Bearer jwt",
				"X-Custom":            "a\tb",
				"X-Empty":             "",
			}},
		},
		{
			name:    "authorization in any case",
			cfg:     RetryConfig{Headers: map[string]string{"authorization": "Bearer x"}},
			wantErr: []string{`"authorization" cannot be set`, "use `token`"},
		},
		{
			name: "every reserved header",
			cfg: RetryConfig{Headers: map[string]string{
				"Accept": "a", "Content-Type": "b", "Content-Length": "1", "Host": "h", "User-Agent": "u",
			}},
			wantErr: []string{`"Accept"`, `"Content-Type"`, `"Content-Length"`, `"Host"`, `"User-Agent"`},
		},
		{
			name:    "invalid name",
			cfg:     RetryConfig{Headers: map[string]string{"X Bad": "v", "": "v"}},
			wantErr: []string{`"X Bad" is not a valid HTTP header name`, `"" is not a valid HTTP header name`},
		},
		{
			name:    "newline in value",
			cfg:     RetryConfig{Headers: map[string]string{"X-Inject": "a\r\nAuthorization: Bearer x"}},
			wantErr: []string{`the value of "X-Inject" contains a control character`},
		},
		{
			name:    "same name twice",
			cfg:     RetryConfig{Headers: map[string]string{"X-Tenant": "a", "x-tenant": "b"}},
			wantErr: []string{`"X-Tenant" and "x-tenant" are the same header`},
		},
		{
			name: "cloudflare headers without the dedicated attributes",
			cfg: RetryConfig{Headers: map[string]string{
				"CF-Access-Client-Id": "id", "CF-Access-Client-Secret": "secret",
			}},
		},
		{
			name: "cloudflare headers with the dedicated attributes",
			cfg: RetryConfig{
				CFAccessClientID:  "id",
				CFAccessClientSec: "secret",
				Headers: map[string]string{
					"cf-access-client-id": "id2", "CF-Access-Client-Secret": "secret2",
				},
			},
			wantErr: []string{"conflicts with `cf_access_client_id`", "conflicts with `cf_access_client_secret`"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.ValidateHeaders()
			if len(tt.wantErr) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, want := range tt.wantErr {
				assert.Contains(t, err.Error(), want)
			}
			for _, value := range tt.cfg.Headers {
				if len(value) > 5 {
					assert.NotContains(t, err.Error(), value, "errors must not quote header values")
				}
			}
		})
	}
}

func TestSameRedirectTarget(t *testing.T) {
	t.Parallel()
	parse := func(raw string) *url.URL {
		t.Helper()
		u, err := url.Parse(raw)
		require.NoError(t, err)
		return u
	}
	assert.True(t, sameRedirectTarget(parse("https://example.com/a"), parse("https://example.com:443/b")))
	assert.True(t, sameRedirectTarget(parse("http://example.com"), parse("http://example.com:80/v")))
	assert.False(t, sameRedirectTarget(parse("http://example.com"), parse("https://example.com")))
	assert.False(t, sameRedirectTarget(parse("https://a.example.com"), parse("https://example.com")))
	assert.False(t, sameRedirectTarget(parse("http://127.0.0.1:1"), parse("http://127.0.0.1:2")))
}

func TestClient_ExtraHeaders_CrossHostRedirectDropsThem(t *testing.T) {
	t.Parallel()
	var firstTenant, destTenant, destCF string
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destTenant = r.Header.Get("X-Tenant")
		destCF = r.Header.Get("CF-Access-Client-Id")
		_ = json.NewEncoder(w).Encode("4.4.0")
	}))
	t.Cleanup(dest.Close)
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstTenant = r.Header.Get("X-Tenant")
		http.Redirect(w, r, dest.URL+"/api/v1/version", http.StatusFound)
	}))
	t.Cleanup(src.Close)

	c := New(src.URL, "test-token", RetryConfig{
		CFAccessClientID: "cf-id",
		Headers:          map[string]string{"X-Tenant": "tenant-a"},
	})
	_, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "tenant-a", firstTenant)
	assert.Empty(t, destTenant)
	assert.Empty(t, destCF)
}

func TestClient_ExtraHeaders_SameHostRedirectKeepsThem(t *testing.T) {
	t.Parallel()
	var got string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			http.Redirect(w, r, srv.URL+"/api/v1/health", http.StatusFound)
			return
		}
		got = r.Header.Get("X-Tenant")
		_, _ = w.Write([]byte("4.4.0"))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-token", RetryConfig{Headers: map[string]string{"X-Tenant": "tenant-a"}})
	_, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "tenant-a", got)
}
