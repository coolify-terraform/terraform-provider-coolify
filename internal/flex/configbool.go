package flex

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConfigBoolTrue reports whether configuration sets attr to true.
//
// Use this for request-only flags Coolify never returns. ignore_changes copies
// null state onto the plan, so the plan is the wrong source after import.
func ConfigBoolTrue(ctx context.Context, config tfsdk.Config, attr path.Path, diags *diag.Diagnostics) bool {
	var v types.Bool
	diags.Append(config.GetAttribute(ctx, attr, &v)...)
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}
