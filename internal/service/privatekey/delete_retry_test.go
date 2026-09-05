package privatekey

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

func TestIsPrivateKeyDeleteRetryable_IgnoresWrapText(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("delete failed: key in use and cannot be deleted: %w",
		&client.APIStatusError{Status: http.StatusInternalServerError, Message: "internal error"})
	if isPrivateKeyDeleteRetryable(err) {
		t.Fatal("must not treat wrap text as retryable")
	}
}

func TestIsPrivateKeyDeleteRetryable_MatchesAPIMessage(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("deleting private key: %w",
		&client.APIStatusError{Status: http.StatusUnprocessableEntity, Message: "This key is in use and cannot be deleted."})
	if !isPrivateKeyDeleteRetryable(err) {
		t.Fatal("expected APIStatusError.Message match")
	}
}
