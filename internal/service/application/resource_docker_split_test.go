package application

import (
	"strings"
	"testing"
)

// digest64 is a stand-in SHA-256 hex digest (64 chars). Coolify stores
// image@sha256:<hex> as name "image@sha256" and tag "<hex>" on update.
var digest64 = strings.Repeat("a", 64)

func TestSplitDockerImage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		image    string
		wantName string
		wantTag  string
	}{
		{
			name:     "name and tag",
			image:    "nginx:latest",
			wantName: "nginx",
			wantTag:  "latest",
		},
		{
			name:     "namespaced image with tag",
			image:    "ghcr.io/org/app:1.25",
			wantName: "ghcr.io/org/app",
			wantTag:  "1.25",
		},
		{
			name:     "no tag",
			image:    "nginx",
			wantName: "nginx",
			wantTag:  "",
		},
		{
			name:     "registry with port is not split at the port colon",
			image:    "localhost:5000/nginx",
			wantName: "localhost:5000/nginx",
			wantTag:  "",
		},
		{
			name:     "empty image",
			image:    "",
			wantName: "",
			wantTag:  "",
		},
		{
			name:     "registry with port and tag",
			image:    "localhost:5000/nginx:1.25",
			wantName: "localhost:5000/nginx",
			wantTag:  "1.25",
		},
		{
			name:     "digest pin",
			image:    "nginx@sha256:" + digest64,
			wantName: "nginx@sha256",
			wantTag:  digest64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, tag := splitDockerImage(tt.image)
			if name != tt.wantName {
				t.Errorf("splitDockerImage(%q) name = %q, want %q", tt.image, name, tt.wantName)
			}
			if tag != tt.wantTag {
				t.Errorf("splitDockerImage(%q) tag = %q, want %q", tt.image, tag, tt.wantTag)
			}
		})
	}
}

// TestSplitDockerImage_UpdateNameHasNoEmbeddedTag fails if splitDockerImage
// returns a name that still looks tagged. Coolify create accepts image:tag
// (DockerImageFormat); PATCH name uses dockerImageNameRules. The Update
// mock in resource_docker_test.go is the HTTP write-path guard.
func TestSplitDockerImage_UpdateNameHasNoEmbeddedTag(t *testing.T) {
	t.Parallel()
	images := []string{
		"nginx:latest",
		"nginx:1.25",
		"ghcr.io/org/app:v1",
		"nginx",
		"localhost:5000/nginx",
		"localhost:5000/nginx:1.25",
		"nginx@sha256:" + digest64,
	}
	for _, image := range images {
		name, _ := splitDockerImage(image)
		if dockerImageNameHasEmbeddedTag(name) {
			t.Errorf("Update would send tagged docker_registry_image_name %q for %q", name, image)
		}
	}
}

func TestDockerImageNameHasEmbeddedTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want bool
	}{
		{in: "nginx:latest", want: true},
		{in: "nginx:1.25", want: true},
		{in: "ghcr.io/org/app:v1", want: true},
		{in: "nginx", want: false},
		{in: "localhost:5000/nginx", want: false},
		{in: "", want: false},
	}
	for _, tt := range tests {
		name := tt.in
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := dockerImageNameHasEmbeddedTag(tt.in); got != tt.want {
				t.Errorf("dockerImageNameHasEmbeddedTag(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
