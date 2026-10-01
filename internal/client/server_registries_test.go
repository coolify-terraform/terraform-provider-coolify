package client

import "testing"

func TestNormalizeDockerRegistry(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"ghcr.io":                     "ghcr.io",
		"https://ghcr.io/v2/":         "ghcr.io",
		"HTTPS://Index.Docker.io/v1/": "docker.io",
		"registry-1.docker.io":        "docker.io",
		"registry.example.com:5000":   "registry.example.com:5000",
		"registry.hub.docker.com":     "docker.io",
	}
	for in, want := range cases {
		if got := NormalizeDockerRegistry(in); got != want {
			t.Errorf("NormalizeDockerRegistry(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestApplyDiskIntervalWrite(t *testing.T) {
	t.Parallel()
	hours := 12
	if got := applyDiskIntervalWrite(&Client{CoolifyVersion: "4.3.23"}, &hours); got != nil {
		t.Fatalf("4.3.23 sent interval %v", *got)
	}
	if got := applyDiskIntervalWrite(&Client{CoolifyVersion: "4.4-rc.1"}, &hours); got != nil {
		t.Fatalf("4.4-rc.1 sent interval %v", *got)
	}
	if got := applyDiskIntervalWrite(&Client{CoolifyVersion: "4.4.0"}, &hours); got == nil || *got != 12 {
		t.Fatalf("4.4.0 interval = %v", got)
	}
	if got := applyDiskIntervalWrite(&Client{}, &hours); got == nil || *got != 12 {
		t.Fatalf("empty version interval = %v", got)
	}
}
