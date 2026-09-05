package applicationpreview

import (
	"errors"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

func TestAnnotatePreviewDomainError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "not found",
			err:  &client.NotFoundError{Message: "resource not found (PATCH /previews/1): Preview not found."},
			want: "Coolify has no preview for this PR yet",
		},
		{
			name: "status 404 text",
			err:  errors.New("api error (status 404) for PATCH /previews/1: Preview not found."),
			want: "Coolify has no preview for this PR yet",
		},
		{
			name: "conflict",
			err:  &client.APIStatusError{Status: 409, Message: "Domain conflict."},
			want: "force_domain_override = true",
		},
		{
			name: "status 409 text",
			err:  errors.New("api error (status 409) for PATCH /previews/1: Domain conflict."),
			want: "force_domain_override = true",
		},
		{
			name: "other",
			err:  errors.New("internal server error"),
			want: "internal server error",
		},
		{
			name: "nil",
			err:  nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := annotatePreviewDomainError(tt.err)
			if !strings.Contains(got, tt.want) {
				t.Errorf("annotatePreviewDomainError() = %q, want substring %q", got, tt.want)
			}
		})
	}
}
