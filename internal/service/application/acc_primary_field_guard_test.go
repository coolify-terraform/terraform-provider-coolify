package application

import (
	"os"
	"strings"
	"testing"
)

// TestApplicationCRUDFiles_UpdatePrimaryField fails if an acc CRUD
// file regresses to a description-only Update. That pattern missed #784
// (docker_image) and would miss the next create/update validator split.
func TestApplicationCRUDFiles_UpdatePrimaryField(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path    string
		needles []string
	}{
		{
			path:    "resource_docker_image_acc_test.go",
			needles: []string{`nginx:1.27-alpine`},
		},
		{
			path:    "resource_public_git_acc_test.go",
			needles: []string{`"8080"`},
		},
		{
			path:    "resource_private_git_acc_test.go",
			needles: []string{`"8080"`},
		},
		{
			path:    "resource_github_app_acc_test.go",
			needles: []string{`"8080"`},
		},
		{
			path:    "resource_dockerfile_acc_test.go",
			needles: []string{`health_check_path = "/ready"`},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			b, err := os.ReadFile(tc.path) //nolint:gosec // test fixture path from the cases table
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			body := string(b)
			for _, needle := range tc.needles {
				if !strings.Contains(body, needle) {
					t.Errorf("%s acc CRUD must update a primary field; missing %q", tc.path, needle)
				}
			}
		})
	}
}
