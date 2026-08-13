package flex_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAddCreateReadBackError(t *testing.T) {
	t.Parallel()
	resp := &resource.CreateResponse{}
	flex.AddCreateReadBackError(resp, "S3 storage", "uuid-1", errors.New("boom"))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostic")
	}
	errs := resp.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if got := errs[0].Summary(); got != "S3 storage created but refresh failed" {
		t.Fatalf("summary: got %q", got)
	}
	detail := errs[0].Detail()
	for _, want := range []string{
		"Coolify created S3 storage uuid-1",
		"Could not read S3 storage uuid-1 after create: boom",
		"partial Terraform state was saved",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q:\n%s", want, detail)
		}
	}
}

func TestAddCreateReadBackNotFoundError(t *testing.T) {
	t.Parallel()
	resp := &resource.CreateResponse{}
	flex.AddCreateReadBackNotFoundError(resp, "application", "uuid-2")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostic")
	}
	errs := resp.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if got := errs[0].Summary(); got != "application created but refresh failed" {
		t.Fatalf("summary: got %q", got)
	}
	detail := errs[0].Detail()
	for _, want := range []string{
		"Coolify created application uuid-2",
		"API returned 404 on the immediate read-back",
		"after the application becomes readable through the API",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q:\n%s", want, detail)
		}
	}
}

func TestCreateReadBackFormatters_MixedCase(t *testing.T) {
	t.Parallel()
	// Applications keep title-case summary and lower-case body label.
	if got := flex.CreateReadBackFailedSummary("Application"); got != "Application created but refresh failed" {
		t.Fatalf("summary: %q", got)
	}
	detail := flex.CreateReadBackFailedDetail("application", "u", errors.New("x"))
	if !strings.Contains(detail, "Coolify created application u") || !strings.Contains(detail, "Could not read application u") {
		t.Fatalf("detail: %s", detail)
	}
}
