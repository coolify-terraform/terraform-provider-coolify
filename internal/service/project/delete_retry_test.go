package project

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

func TestIsProjectDeleteRetryable_IgnoresWrapText(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("could not delete project that has resources: %w",
		&client.APIStatusError{Status: http.StatusInternalServerError, Message: "internal error"})
	if isProjectDeleteRetryable(err) {
		t.Fatal("must not treat wrap text as retryable")
	}
}

func TestIsProjectDeleteRetryable_MatchesAPIMessage(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("deleting project: %w",
		&client.APIStatusError{Status: http.StatusUnprocessableEntity, Message: "Project has resources, so it cannot be deleted."})
	if !isProjectDeleteRetryable(err) {
		t.Fatal("expected APIStatusError.Message match")
	}
}
