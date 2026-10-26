package client

import "testing"

func TestSupportsServiceApplicationWriteGates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		version string
		force   bool
		restart bool
	}{
		{version: "", force: true, restart: true},
		{version: "4.3.0", force: true, restart: true},
		{version: "4.3.7", force: false, restart: false},
		{version: "4.3.8", force: true, restart: false},
		{version: "4.3.18", force: true, restart: false},
		{version: "4.3.19", force: true, restart: true},
		{version: "4.4-rc.1", force: true, restart: false},
		{version: "v4.4-rc.1", force: true, restart: false},
		{version: "4.4.0", force: true, restart: true},
		{version: "4.4.2", force: true, restart: true},
		{version: "4.5.0", force: true, restart: true},
	}
	for _, tc := range cases {
		c := &Client{CoolifyVersion: tc.version}
		if got := c.SupportsServiceApplicationForceHTTPS(); got != tc.force {
			t.Errorf("%s force https = %v, want %v", tc.version, got, tc.force)
		}
		if got := c.SupportsServiceApplicationMaxRestart(); got != tc.restart {
			t.Errorf("%s max restart = %v, want %v", tc.version, got, tc.restart)
		}
	}
}

func TestSupportsServiceApplicationRoute(t *testing.T) {
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
		if got := c.SupportsServiceApplicationRoute(); got != tc.want {
			t.Errorf("%s route = %v, want %v", tc.version, got, tc.want)
		}
	}
}
