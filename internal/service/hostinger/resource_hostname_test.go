package hostinger

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWarnHostingerHostname(t *testing.T) {
	t.Parallel()

	var single diag.Diagnostics
	warnHostingerHostname(types.StringValue("app-node"), &single)
	if got := warningCount(single); got != 1 {
		t.Fatalf("warning count = %d, want 1", got)
	}
	if summary := single[0].Summary(); summary != "Hostinger will choose the VPS hostname" {
		t.Fatalf("summary = %q", summary)
	}

	var dotted diag.Diagnostics
	warnHostingerHostname(types.StringValue("app.example.com"), &dotted)
	if got := warningCount(dotted); got != 0 {
		t.Fatalf("dotted name warnings = %d, want 0", got)
	}

	var empty diag.Diagnostics
	warnHostingerHostname(types.StringNull(), &empty)
	warnHostingerHostname(types.StringUnknown(), &empty)
	if got := warningCount(empty); got != 0 {
		t.Fatalf("null or unknown warnings = %d, want 0", got)
	}
}

func warningCount(d diag.Diagnostics) int {
	n := 0
	for _, item := range d {
		if item.Severity() == diag.SeverityWarning {
			n++
		}
	}
	return n
}
