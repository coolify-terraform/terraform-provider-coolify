package client

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// reservedHeaders are set by the provider or by Go's HTTP client on every
// request. A user header with one of these names would be replaced or
// ignored, so ValidateHeaders rejects it instead.
var reservedHeaders = map[string]string{
	"Accept":         "the provider sets it on every request",
	"Authorization":  "the provider sends the Coolify API token in it; use `token`",
	"Content-Length": "the HTTP client computes it",
	"Content-Type":   "the provider sets it on every request with a body",
	"Host":           "the HTTP client sets it from `endpoint`",
	"User-Agent":     "the provider sets it on every request",
}

// ValidateHeaders checks the user headers in cfg.Headers. It rejects
// invalid names and values, two names that differ only in case, the
// headers the provider or the HTTP client set themselves, and the
// Cloudflare Access headers when their dedicated settings are used.
func (cfg RetryConfig) ValidateHeaders() error {
	names := make([]string, 0, len(cfg.Headers))
	for name := range cfg.Headers {
		names = append(names, name)
	}
	sort.Strings(names)

	var problems []string
	seen := make(map[string]string, len(names))
	for _, name := range names {
		if problem := cfg.headerProblem(name, cfg.Headers[name]); problem != "" {
			problems = append(problems, problem)
			continue
		}
		canonical := http.CanonicalHeaderKey(name)
		if other, ok := seen[canonical]; ok {
			problems = append(problems, fmt.Sprintf("%q and %q are the same header: HTTP header names are case-insensitive", other, name))
			continue
		}
		seen[canonical] = name
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

// headerProblem describes what is wrong with one header, or returns "".
func (cfg RetryConfig) headerProblem(name, value string) string {
	if !validHeaderName(name) {
		return fmt.Sprintf("%q is not a valid HTTP header name", name)
	}
	if !validHeaderValue(value) {
		return fmt.Sprintf("the value of %q contains a control character such as a newline", name)
	}
	canonical := http.CanonicalHeaderKey(name)
	if reason, ok := reservedHeaders[canonical]; ok {
		return fmt.Sprintf("%q cannot be set: %s", name, reason)
	}
	switch {
	case canonical == "Cf-Access-Client-Id" && cfg.CFAccessClientID != "":
		return fmt.Sprintf("%q conflicts with `cf_access_client_id`: set it in one place only", name)
	case canonical == "Cf-Access-Client-Secret" && cfg.CFAccessClientSec != "":
		return fmt.Sprintf("%q conflicts with `cf_access_client_secret`: set it in one place only", name)
	}
	return ""
}

// validHeaderName reports whether name is an RFC 9110 token.
func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0:
		default:
			return false
		}
	}
	return true
}

// validHeaderValue rejects control characters other than horizontal tab,
// which would let a value inject another header or break the request.
func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}
	return true
}
