package service_test

import (
	"os"
	"strings"
	"testing"
)

// TestServiceCRUDFiles_UpdatePrimaryField fails if service acc regresses
// to Create+Import with no name Update. That pattern never exercised
// ServicesController::update_by_uuid extra-key 422.
func TestServiceCRUDFiles_UpdatePrimaryField(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("resource_acc_test.go")
	if err != nil {
		t.Fatalf("read resource_acc_test.go: %v", err)
	}
	body := string(b)
	if !strings.Contains(body, `name + "-upd"`) && !strings.Contains(body, `"-upd"`) {
		t.Error("service acc CRUD must update name (PATCH allow-listed primary field)")
	}
}
