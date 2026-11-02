package client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

const defaultAttemptTimeout = 30 * time.Second

// failureTransport fails every request with a configuration error.
// New uses it when the TLS material cannot be parsed, so a direct
// client.New caller still surfaces the error on the first call.
type failureTransport struct{ err error }

func (f failureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

// newTransport builds the HTTP transport for every client.
// It starts from cleanhttp.DefaultPooledTransport so ProxyFromEnvironment
// and HTTP/2 stay in place, including when ca_cert or insecure is set.
// A non-nil cfg.Proxy replaces the environment proxy.
func newTransport(cfg RetryConfig) (http.RoundTripper, error) {
	base := cleanhttp.DefaultPooledTransport()
	tlsCfg, err := tlsConfig(cfg)
	if err != nil {
		return nil, err
	}
	if tlsCfg != nil {
		base.TLSClientConfig = tlsCfg
	}
	if cfg.Proxy != nil {
		base.Proxy = cfg.Proxy
	}
	return base, nil
}

// attemptTimeout is the per-attempt deadline. Zero means the default.
func attemptTimeout(cfg RetryConfig) time.Duration {
	if cfg.RequestTimeout > 0 {
		return cfg.RequestTimeout
	}
	return defaultAttemptTimeout
}

// EncryptedPrivateKey reports whether pemText is an encrypted PKCS#8
// or traditional encrypted PEM private key. tls.X509KeyPair does not
// decrypt those, and x509.DecryptPEMBlock is deprecated.
func EncryptedPrivateKey(pemText string) bool {
	if strings.Contains(pemText, "ENCRYPTED PRIVATE KEY") {
		return true
	}
	return strings.Contains(pemText, "Proc-Type: 4,ENCRYPTED")
}

// ValidateRetryConfig checks TLS material that New would reject.
// A nil error with no custom TLS settings is success.
func ValidateRetryConfig(cfg RetryConfig) error {
	_, err := tlsConfig(cfg)
	return err
}

// tlsConfig returns nil when the client should keep the default TLS
// settings (system roots, certificate name from the URL).
func tlsConfig(cfg RetryConfig) (*tls.Config, error) {
	if !customTLS(cfg) {
		return nil, nil
	}
	if err := clientCertPairError(cfg); err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{ //nolint:gosec // InsecureSkipVerify is set only when the user opted in
		InsecureSkipVerify: cfg.Insecure,
		ServerName:         cfg.TLSServerName,
	}
	if cfg.CACert != "" {
		pool, err := trustedRoots(cfg.CACert)
		if err != nil {
			return nil, err
		}
		tlsCfg.RootCAs = pool
	}
	if cfg.ClientCert != "" {
		cert, err := tls.X509KeyPair([]byte(cfg.ClientCert), []byte(cfg.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("parsing client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

func customTLS(cfg RetryConfig) bool {
	return cfg.CACert != "" || cfg.Insecure || cfg.ClientCert != "" || cfg.ClientKey != "" || cfg.TLSServerName != ""
}

func clientCertPairError(cfg RetryConfig) error {
	if (cfg.ClientCert == "") != (cfg.ClientKey == "") {
		missing := "client_key"
		if cfg.ClientKey != "" {
			missing = "client_cert"
		}
		return fmt.Errorf("%s must be set when the other client certificate attribute is set", missing)
	}
	if cfg.ClientKey != "" && EncryptedPrivateKey(cfg.ClientKey) {
		return errors.New("client_key is an encrypted private key; provide an unencrypted PKCS#8 or PKCS#1 key")
	}
	return nil
}

func trustedRoots(pemText string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM([]byte(pemText)) {
		return nil, errors.New("ca_cert is not a PEM-encoded certificate")
	}
	return pool, nil
}
