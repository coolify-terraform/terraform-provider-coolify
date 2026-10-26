package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupportsCloudProviderCatalog(t *testing.T) {
	t.Parallel()
	cases := []struct {
		version string
		want    bool
	}{
		{version: "", want: true},
		{version: "4.1.2", want: false},
		{version: "v4.1.2", want: false},
		{version: "4.2.0", want: true},
		{version: "4.3.0", want: true},
		{version: "4.4-rc.1", want: true},
		{version: "4.5.0", want: true},
	}
	for _, tc := range cases {
		c := &Client{CoolifyVersion: tc.version}
		if got := c.SupportsCloudProviderCatalog(); got != tc.want {
			t.Errorf("%s catalog = %v, want %v", tc.version, got, tc.want)
		}
	}
}

func TestCloudCatalog_OldCoolifySkipsHTTP(t *testing.T) {
	t.Parallel()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.1.2"
	ctx := context.Background()

	if _, err := c.ListDigitalOceanRegions(ctx, "tok"); err == nil || !strings.Contains(err.Error(), "v4.2.0") {
		t.Fatalf("regions: %v", err)
	}
	if _, err := c.ListHetznerLocations(ctx, "tok"); err == nil || !strings.Contains(err.Error(), "v4.2.0") {
		t.Fatalf("locations: %v", err)
	}
	if _, err := c.ListVultrPlans(ctx, "tok"); err == nil || !strings.Contains(err.Error(), "v4.2.0") {
		t.Fatalf("plans: %v", err)
	}
	if _, err := c.CreateHetznerServer(ctx, CreateHetznerServerInput{}); err == nil || !strings.Contains(err.Error(), "v4.2.0") {
		t.Fatalf("hetzner create: %v", err)
	}
	if _, err := c.CreateDigitalOceanServer(ctx, CreateDigitalOceanServerInput{}); err == nil || !strings.Contains(err.Error(), "v4.2.0") {
		t.Fatalf("digitalocean create: %v", err)
	}
	if _, err := c.CreateVultrServer(ctx, CreateVultrServerInput{}); err == nil || !strings.Contains(err.Error(), "v4.2.0") {
		t.Fatalf("vultr create: %v", err)
	}
	if called {
		t.Fatal("old Coolify called a cloud catalog route")
	}
}
