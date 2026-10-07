package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactEndpointForDiagnostics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no userinfo", "https://coolify.example.com/api", "https://coolify.example.com/api"},
		{"username only", "https://user@coolify.example.com", "https://REDACTED@coolify.example.com"},
		{"username and password", "https://user:pass@coolify.example.com/api", "https://REDACTED:REDACTED@coolify.example.com/api"},
		{"invalid url", "://not a url", "://not a url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, redactEndpointForDiagnostics(tt.input))
		})
	}
}

func TestHTTPEndpointWarnsCleartext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		warns bool
	}{
		{"bracketed ipv6 loopback with port", "http://[::1]:8000", false},
		{"bracketed ipv6 loopback", "http://[::1]", false},
		{"userinfo on loopback", "http://user:pass@127.0.0.1:8000", false},
		{"loopback with path", "http://127.0.0.1:8000/api", false},
		{"localhost", "http://localhost:8000", false},
		{"public name", "http://example.com:8000", true},
		{"non-loopback ipv6", "http://[2001:db8::1]:8000", true},
		{"https loopback", "https://[::1]:8000", false},
		{"unparsable http", "http://[::1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.warns, httpEndpointWarnsCleartext(tt.input))
		})
	}
}

func TestIsVersionAtLeast(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		actual   string
		minimum  string
		expected bool
	}{
		{"equal", "4.0.0", "4.0.0", true},
		{"higher major", "5.0.0", "4.0.0", true},
		{"higher minor", "4.1.0", "4.0.0", true},
		{"higher patch", "4.0.1", "4.0.0", true},
		{"lower major", "3.9.9", "4.0.0", false},
		{"lower minor", "4.0.0", "4.1.0", false},
		{"lower patch", "4.0.0", "4.0.1", false},
		{"v prefix actual", "v4.0.0", "4.0.0", true},
		{"v prefix minimum", "4.0.0", "v4.0.0", true},
		{"v prefix both", "v4.1.0", "v4.0.0", true},
		{"pre-release suffix", "4.0.0-beta.335", "4.0.0", true},
		{"pre-release lower", "3.9.0-beta.1", "4.0.0", false},
		{"two-part version", "4.1", "4.0.0", true},
		{"two-part lower", "3.9", "4.0.0", false},
		{"garbage actual", "latest", "4.0.0", true},
		{"empty actual", "", "4.0.0", true},
		{"garbage minimum", "4.0.0", "latest", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isVersionAtLeast(tt.actual, tt.minimum))
		})
	}
}
