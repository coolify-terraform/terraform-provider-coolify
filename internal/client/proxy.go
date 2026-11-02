package client

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/http/httpproxy"
)

// ProxySettings is an explicit forward proxy. An empty URL means the
// process environment (HTTP_PROXY, HTTPS_PROXY, NO_PROXY) stays in effect.
type ProxySettings struct {
	URL      string
	Username string
	Password string
	// NoProxySet reports that no_proxy was configured, including as an
	// empty string. An empty value replaces NO_PROXY. When NoProxySet
	// is false, the process NO_PROXY and no_proxy variables still apply.
	NoProxySet bool
	NoProxy    string
}

// ProxyFunction builds a transport proxy from s.
// A nil function means "keep ProxyFromEnvironment".
// Error text never includes the proxy password.
func ProxyFunction(s ProxySettings) (func(*http.Request) (*url.URL, error), error) {
	if s.URL == "" && s.Username == "" && s.Password == "" && !s.NoProxySet {
		return nil, nil
	}
	if s.URL == "" {
		return nil, fmt.Errorf("proxy url is required when proxy username, password, or no_proxy is set")
	}
	parsed, err := url.Parse(s.URL)
	if err != nil || parsed.Host == "" || parsed.Scheme == "" {
		return nil, fmt.Errorf("proxy url must be an absolute http, socks5, or socks5h URL")
	}
	switch parsed.Scheme {
	case "http", "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("proxy url scheme %q is not supported; use http, socks5, or socks5h", parsed.Scheme)
	}
	if parsed.User != nil && (s.Username != "" || s.Password != "") {
		return nil, fmt.Errorf("set proxy username and password as attributes, or in the proxy URL, not both")
	}
	if s.Username != "" || s.Password != "" {
		parsed.User = url.UserPassword(s.Username, s.Password)
	}
	noProxy := environmentNoProxy()
	if s.NoProxySet {
		noProxy = s.NoProxy
	}
	// String() includes userinfo. httpproxy returns that URL to the
	// transport, which sends it as proxy basic auth. Callers must not
	// log the returned URL.
	proxyURL := parsed.String()
	matcher := (&httpproxy.Config{
		HTTPProxy:  proxyURL,
		HTTPSProxy: proxyURL,
		NoProxy:    noProxy,
	}).ProxyFunc()
	return func(req *http.Request) (*url.URL, error) {
		if req == nil || req.URL == nil {
			return nil, fmt.Errorf("proxy: request URL is missing")
		}
		return matcher(req.URL)
	}, nil
}

func environmentNoProxy() string {
	if v, ok := os.LookupEnv("NO_PROXY"); ok {
		return v
	}
	return os.Getenv("no_proxy")
}
