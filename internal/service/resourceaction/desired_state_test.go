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

func TestIsAlreadyInDesiredState_DatabaseStartInProgress(t *testing.T) {
	t.Parallel()
	const msg = "Another start, restart or import of this database is already in progress."
	err := fmt.Errorf("starting database: %w",
		&client.APIStatusError{Status: http.StatusConflict, Message: msg})
	if !isAlreadyInDesiredState(err, "start") {
		t.Fatal("expected 409 in-progress database start to be idempotent")
	}
	if isAlreadyInDesiredState(err, "restart") {
		t.Fatal("restart must still fail when Coolify rejects it with 409")
	}
	if isAlreadyInDesiredState(err, "stop") {
		t.Fatal("stop must still fail when the API returns 409")
	}

	// The phrase in an outer wrap is not the API message.
	wrapped := fmt.Errorf("already in progress: %w",
		&client.APIStatusError{Status: http.StatusConflict, Message: "Domain conflicts detected."})
	if isAlreadyInDesiredState(wrapped, "start") {
		t.Fatal("must not treat an unrelated 409 as an in-progress start")
	}

	badRequest := fmt.Errorf("starting database: %w",
		&client.APIStatusError{Status: http.StatusBadRequest, Message: msg})
	if isAlreadyInDesiredState(badRequest, "start") {
		t.Fatal("in-progress handling is HTTP 409 only")
	}
}
