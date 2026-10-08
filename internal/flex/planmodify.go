package flex

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// RequiresReplaceIfKnown requires replacement when a stored value changes.
// A null or unknown state adopts the configuration instead. Simple UUID import
// leaves project_uuid, server_uuid, and destination_uuid null because Coolify
// GET does not return them. RequiresReplace would destroy the resource when
// the user sets those fields in configuration.
func RequiresReplaceIfKnown() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
				resp.RequiresReplace = false
				return
			}
			resp.RequiresReplace = true
		},
		"Changing a known value forces a new resource. A null or unknown value adopts the configuration.",
		"Changing a known value forces a new resource. A null or unknown value adopts the configuration.",
	)
}

// BoolRequiresReplaceIfKnown is RequiresReplaceIfKnown for bool attributes.
// Create-only cloud flags are not returned by GET. A null state after import
// must adopt the configuration. A change from a known value still replaces
// the resource, because Coolify cannot update the flag.
func BoolRequiresReplaceIfKnown() planmodifier.Bool {
	return boolplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
			// Null or unknown state is an import. Null or unknown config is
			// an omitted flag: the plan may still be unknown when this runs,
			// before UseStateForUnknown copies state. Neither case is a change.
			if req.StateValue.IsNull() || req.StateValue.IsUnknown() ||
				req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
				resp.RequiresReplace = false
				return
			}
			resp.RequiresReplace = true
		},
		"Changing a known value forces a new resource. A null or unknown value adopts the configuration.",
		"Changing a known value forces a new resource. A null or unknown value adopts the configuration.",
	)
}
