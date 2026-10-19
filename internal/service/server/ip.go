package server

import (
	"net/netip"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NormalizeServerIP matches Coolify Server::normalizeIp on v4.4.6 and later.
// The setter drops characters outside [0-9A-Za-z.:%-], then compresses IPv6
// to the short lowercase form. IPv4 and hostnames are otherwise unchanged.
func NormalizeServerIP(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if isServerIPChar(r) {
			b.WriteRune(r)
		}
	}
	return normalizeIPAddress(b.String())
}

func isServerIPChar(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		r == '.' || r == ':' || r == '%' || r == '-'
}

// normalizeIPAddress matches Coolify normalizeIpAddress. Brackets are removed
// only when the inside is a real IPv6 address. A zone id is left alone:
// PHP's IPv6 filter rejects it.
func normalizeIPAddress(value string) string {
	value = strings.TrimSpace(value)
	withoutBrackets := strings.Trim(value, "[]")
	addr, err := netip.ParseAddr(withoutBrackets)
	if err != nil || !addr.Is6() || addr.Zone() != "" {
		return value
	}
	return addr.String()
}

// keepEquivalentServerIP returns the config or state text when Coolify's
// stored form is the same address. A different address uses the API value.
func keepEquivalentServerIP(current types.String, api string) string {
	if current.IsNull() || current.IsUnknown() {
		return api
	}
	user := current.ValueString()
	if user == api || NormalizeServerIP(user) == NormalizeServerIP(api) {
		return user
	}
	return api
}
