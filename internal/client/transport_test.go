package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttemptTimeout_Default(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 30*time.Second, attemptTimeout(RetryConfig{}))
	assert.Equal(t, 45*time.Second, attemptTimeout(RetryConfig{RequestTimeout: 45 * time.Second}))
}

func TestNewTransport_KeepsEnvironmentProxyAndHTTP2(t *testing.T) {
	t.Parallel()
	cases := []RetryConfig{
		{},
		{CACert: string(mustSelfSignedPEM(t))},
		{Insecure: true},
		{TLSServerName: "coolify.internal"},
	}
	for _, cfg := range cases {
		tr, err := newTransport(cfg)
		require.NoError(t, err)
		httpTr, ok := tr.(*http.Transport)
		require.True(t, ok)
		assert.Equal(t, reflect.ValueOf(http.ProxyFromEnvironment).Pointer(), reflect.ValueOf(httpTr.Proxy).Pointer())
		assert.True(t, httpTr.ForceAttemptHTTP2)
	}
}

func TestNewTransport_InjectedProxyReplacesEnvironment(t *testing.T) {
	t.Parallel()
	proxyURL, err := url.Parse("http://proxy.example:8080")
	require.NoError(t, err)
	tr, err := newTransport(RetryConfig{
		CACert: string(mustSelfSignedPEM(t)),
		Proxy: func(*http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
	})
	require.NoError(t, err)
	httpTr := tr.(*http.Transport)
	got, err := httpTr.Proxy(httptest.NewRequest(http.MethodGet, "https://coolify.example/api", nil))
	require.NoError(t, err)
	assert.Equal(t, proxyURL.String(), got.String())
	assert.NotEqual(t, reflect.ValueOf(http.ProxyFromEnvironment).Pointer(), reflect.ValueOf(httpTr.Proxy).Pointer())
}

func TestClient_CustomCAUsesInjectedProxy(t *testing.T) {
	t.Parallel()
	certs := newTestCA(t)
	var gotAuth, gotProxyAuth string
	var mu sync.Mutex
	origin := tlsServer(t, certs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		gotProxyAuth = r.Header.Get("Proxy-Authorization")
		mu.Unlock()
		_, _ = w.Write([]byte("4.0.0"))
	}))
	connectAuth := &lockedString{}
	proxy := httptest.NewServer(connectTunnel(t, origin.Listener.Addr().String(), connectAuth))
	t.Cleanup(proxy.Close)
	proxyParsed, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	proxyParsed.User = url.UserPassword("proxyuser", "proxypass")

	c := New("https://example.com", "test-token", RetryConfig{
		CACert: certs.caPEM,
		Proxy: func(*http.Request) (*url.URL, error) {
			return proxyParsed, nil
		},
		Headers: map[string]string{"Proxy-Authorization": "Bearer iap-token"},
	})
	version, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "Bearer test-token", gotAuth)
	assert.Equal(t, "Bearer iap-token", gotProxyAuth)
	recorded := connectAuth.get()
	assert.True(t, strings.HasPrefix(recorded, "Basic "), "CONNECT proxy auth was %q", recorded)
	assert.NotContains(t, recorded, "iap-token")
	assert.NotContains(t, recorded, "proxypass")
}

func TestClient_InsecureUsesInjectedProxy(t *testing.T) {
	t.Parallel()
	certs := newTestCA(t)
	origin := tlsServer(t, certs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("4.0.0"))
	}))
	connectAuth := &lockedString{}
	proxy := httptest.NewServer(connectTunnel(t, origin.Listener.Addr().String(), connectAuth))
	t.Cleanup(proxy.Close)
	proxyParsed, err := url.Parse(proxy.URL)
	require.NoError(t, err)

	c := New("https://example.com", "test-token", RetryConfig{
		Insecure: true,
		Proxy: func(*http.Request) (*url.URL, error) {
			return proxyParsed, nil
		},
	})
	version, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
	assert.NotEmpty(t, connectAuth.get())
}

func TestClient_RequestTimeoutBoundsAttempts(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-token", RetryConfig{
		RequestTimeout: 100 * time.Millisecond,
		Attempts:       1,
		MinWait:        time.Millisecond,
		MaxWait:        time.Millisecond,
	})
	start := time.Now()
	err := c.do(context.Background(), http.MethodGet, "/slow", nil, nil)
	elapsed := time.Since(start)
	require.Error(t, err)
	assert.Equal(t, int32(2), calls.Load(), "one retry plus the first attempt")
	assert.Less(t, elapsed, 5*time.Second)
}

func TestClient_TLSServerName(t *testing.T) {
	t.Parallel()
	certs := newTestCA(t)
	origin := tlsServer(t, certs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("4.0.0"))
	}))
	endpoint := "https://" + origin.Listener.Addr().String()

	c := New(endpoint, "test-token", RetryConfig{
		CACert:        certs.caPEM,
		TLSServerName: "example.com",
	})
	version, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)

	tr, err := newTransport(RetryConfig{CACert: certs.caPEM, TLSServerName: "example.com"})
	require.NoError(t, err)
	assert.Equal(t, reflect.ValueOf(http.ProxyFromEnvironment).Pointer(), reflect.ValueOf(tr.(*http.Transport).Proxy).Pointer())

	wrong := New(endpoint, "test-token", RetryConfig{
		CACert:        certs.caPEM,
		TLSServerName: "wrong.test",
	})
	_, err = wrong.GetVersion(context.Background())
	require.Error(t, err)

	insecure := New(endpoint, "test-token", RetryConfig{
		Insecure:      true,
		TLSServerName: "wrong.test",
	})
	version, err = insecure.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
}

func TestClient_MutualTLS(t *testing.T) {
	t.Parallel()
	certs := newTestCA(t)
	var peerCN lockedString
	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			peerCN.set(r.TLS.PeerCertificates[0].Subject.CommonName)
		}
		_, _ = w.Write([]byte("4.0.0"))
	}))
	origin.TLS = &tls.Config{
		Certificates: []tls.Certificate{certs.server},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certs.pool,
		MinVersion:   tls.VersionTLS12,
	}
	origin.StartTLS()
	t.Cleanup(origin.Close)

	c := New(origin.URL, "test-token", RetryConfig{
		CACert:     certs.caPEM,
		ClientCert: certs.clientCertPEM,
		ClientKey:  certs.clientKeyPEM,
	})
	version, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
	assert.Equal(t, "coolify-client", peerCN.get())

	missing := New(origin.URL, "test-token", RetryConfig{CACert: certs.caPEM})
	_, err = missing.GetVersion(context.Background())
	require.Error(t, err)
}

func TestNewTransport_ClientCertificateErrors(t *testing.T) {
	t.Parallel()
	_, err := newTransport(RetryConfig{ClientCert: "cert"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_key")

	_, err = newTransport(RetryConfig{ClientKey: "key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_cert")

	encrypted := "-----BEGIN ENCRYPTED PRIVATE KEY-----\nAAAA\n-----END ENCRYPTED PRIVATE KEY-----\n"
	_, err = newTransport(RetryConfig{ClientCert: "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n", ClientKey: encrypted})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted")
	assert.NotContains(t, err.Error(), "AAAA")

	certs := newTestCA(t)
	other := newTestCA(t)
	_, err = newTransport(RetryConfig{ClientCert: certs.clientCertPEM, ClientKey: other.clientKeyPEM})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "BEGIN")
}

func TestProxyFunction_NoProxyAndRedaction(t *testing.T) {
	t.Parallel()
	fn, err := ProxyFunction(ProxySettings{URL: "http://proxy.example:8080", NoProxySet: true, NoProxy: "coolify.internal"})
	require.NoError(t, err)
	bypass, err := fn(httptest.NewRequest(http.MethodGet, "https://coolify.internal/api", nil))
	require.NoError(t, err)
	assert.Nil(t, bypass)
	used, err := fn(httptest.NewRequest(http.MethodGet, "https://other.example/api", nil))
	require.NoError(t, err)
	require.NotNil(t, used)
	assert.Equal(t, "proxy.example:8080", used.Host)

	_, err = ProxyFunction(ProxySettings{URL: "https://user:s3cr3t-value@proxy.example:8443"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "s3cr3t-value")
	assert.Contains(t, err.Error(), "https")

	_, err = ProxyFunction(ProxySettings{URL: "http://user:pw@proxy.example:8080", Username: "other"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "pw")

	fn, err = ProxyFunction(ProxySettings{URL: "socks5://proxy.example:1080", Username: "u", Password: "s3cr3t-value"})
	require.NoError(t, err)
	got, err := fn(httptest.NewRequest(http.MethodGet, "https://coolify.example/api", nil))
	require.NoError(t, err)
	assert.Equal(t, "socks5", got.Scheme)
	password, ok := got.User.Password()
	require.True(t, ok)
	assert.Equal(t, "s3cr3t-value", password)

	empty, err := ProxyFunction(ProxySettings{})
	require.NoError(t, err)
	assert.Nil(t, empty)

	_, err = ProxyFunction(ProxySettings{Username: "only-user"})
	require.Error(t, err)
}

type testCA struct {
	caPEM         string
	pool          *x509.CertPool
	server        tls.Certificate
	clientCertPEM string
	clientKeyPEM  string
}

func newTestCA(t *testing.T) testCA {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, caKey.Public(), caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	server := leafCert(t, "example.com", caCert, caKey, false)
	client := leafCert(t, "coolify-client", caCert, caKey, true)
	clientPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: client.Certificate[0]})
	keyDER, err := x509.MarshalPKCS8PrivateKey(client.PrivateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return testCA{
		caPEM:         string(caPEM),
		pool:          pool,
		server:        server,
		clientCertPEM: string(clientPEM),
		clientKeyPEM:  string(keyPEM),
	}
}

func leafCert(t *testing.T, name string, ca *x509.Certificate, caKey *ecdsa.PrivateKey, client bool) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	if client {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.DNSNames = []string{name}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, key.Public(), caKey)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func tlsServer(t *testing.T, certs testCA, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{certs.server},
		MinVersion:   tls.VersionTLS12,
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

type lockedString struct {
	mu sync.Mutex
	v  string
}

func (s *lockedString) set(v string) {
	s.mu.Lock()
	s.v = v
	s.mu.Unlock()
}

func (s *lockedString) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.v
}

func connectTunnel(t *testing.T, target string, connectAuth *lockedString) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "want CONNECT", http.StatusMethodNotAllowed)
			return
		}
		recorded := r.Header.Get("Proxy-Authorization")
		if recorded == "" {
			recorded = "connected"
		}
		connectAuth.set(recorded)
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack", http.StatusInternalServerError)
			return
		}
		client, _, err := hj.Hijack()
		if err != nil {
			return
		}
		defer client.Close()
		upstream, err := net.Dial("tcp", target)
		if err != nil {
			return
		}
		defer upstream.Close()
		if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
		errc := make(chan struct{}, 2)
		go func() {
			_, _ = copyStream(upstream, client)
			errc <- struct{}{}
		}()
		go func() {
			_, _ = copyStream(client, upstream)
			errc <- struct{}{}
		}()
		<-errc
	})
}

func copyStream(dst, src net.Conn) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, werr := dst.Write(buf[:n])
			total += int64(written)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			return total, err
		}
	}
}

func mustSelfSignedPEM(t *testing.T) []byte {
	t.Helper()
	return []byte(newTestCA(t).caPEM)
}
