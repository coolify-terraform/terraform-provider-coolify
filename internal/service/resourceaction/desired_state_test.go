package resourceaction

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

func TestIsAlreadyInDesiredState_IgnoresWrapText(t *testing.T) {
	t.Parallel()
	// Outer wrap contains the phrase; the API message does not.
	err := fmt.Errorf("could not stop already stopped database: %w",
		&client.APIStatusError{Status: http.StatusBadRequest, Message: "validation failed"})
	if isAlreadyInDesiredState(err, "stop") {
		t.Fatal("must not treat wrap text as already-in-desired-state")
	}
}

func TestIsAlreadyInDesiredState_MatchesAPIMessage(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("stopping database: %w",
		&client.APIStatusError{Status: http.StatusBadRequest, Message: "Database is already stopped."})
	if !isAlreadyInDesiredState(err, "stop") {
		t.Fatal("expected APIStatusError.Message match for already stopped")
	}
	err = fmt.Errorf("starting service: %w",
		&client.APIStatusError{Status: http.StatusBadRequest, Message: "Service is already running."})
	if !isAlreadyInDesiredState(err, "start") {
		t.Fatal("expected APIStatusError.Message match for already running")
	}
	err = fmt.Errorf("restarting: %w",
		&client.APIStatusError{Status: http.StatusBadRequest, Message: "Service is already running."})
	if isAlreadyInDesiredState(err, "restart") {
		t.Fatal("restart is not an idempotent desired-state action")
	}
}
