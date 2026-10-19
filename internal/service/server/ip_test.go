package server

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeServerIP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "ipv4", in: "188.245.12.182", want: "188.245.12.182"},
		{name: "ipv4 spaces", in: " 188.245.12.182 ", want: "188.245.12.182"},
		{name: "hostname", in: "server.example.com", want: "server.example.com"},
		{name: "ipv6 short", in: "2a01:4f8:c016:bd23::1", want: "2a01:4f8:c016:bd23::1"},
		{name: "ipv6 long upper", in: "2A01:04F8:C016:BD23:0000:0000:0000:0001", want: "2a01:4f8:c016:bd23::1"},
		{name: "ipv6 compressed upper", in: "2A01:04F8::0001", want: "2a01:4f8::1"},
		{name: "ipv6 brackets", in: "[2a01:4f8:c016:bd23::1]", want: "2a01:4f8:c016:bd23::1"},
		{name: "brackets that are not an ip", in: "[not-an-ip]", want: "not-an-ip"},
		{name: "zone left alone", in: "fe80::1%eth0", want: "fe80::1%eth0"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeServerIP(tt.in); got != tt.want {
				t.Fatalf("NormalizeServerIP(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFlattenServerCommon_KeepsEquivalentIPv6(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	const configured = "2A01:04F8::0001"
	m.IP = types.StringValue(configured)
	FlattenServerCommon(&client.Server{IP: "2a01:4f8::1"}, ptrs)
	if got := m.IP.ValueString(); got != configured {
		t.Fatalf("ip = %q, want config text %q", got, configured)
	}
}

func TestFlattenServerCommon_ReplacesDifferentIP(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	m.IP = types.StringValue("10.0.0.1")
	FlattenServerCommon(&client.Server{IP: "10.0.0.2"}, ptrs)
	if got := m.IP.ValueString(); got != "10.0.0.2" {
		t.Fatalf("ip = %q, want 10.0.0.2", got)
	}
}
