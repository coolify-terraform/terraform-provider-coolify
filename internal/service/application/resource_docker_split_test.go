package application

import (
	"testing"
)

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
