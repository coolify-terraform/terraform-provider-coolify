package flex

import (
	"context"

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
