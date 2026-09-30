package flex

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnvironmentNamePlan defaults a missing environment_name to production unless
// environment_uuid is set. Coolify looks up the name first, so a default name
// plus a UUID would ignore the UUID. Setting both is rejected. A change from
// one known name to another forces a new resource.
func EnvironmentNamePlan() planmodifier.String {
	return environmentNameModifier{}
}

type environmentNameModifier struct{}

func (environmentNameModifier) Description(context.Context) string {
	return "Defaults environment_name to production unless environment_uuid is set. Setting both is an error. Changing a known name forces a new resource."
}

func (m environmentNameModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (environmentNameModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Resource destroy plans a null value. Leave it null. Filling the name
	// here would turn the destroy plan into an object.
	if req.Plan.Raw.IsNull() {
		return
	}
	var nameCfg, uuidCfg types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("environment_name"), &nameCfg)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("environment_uuid"), &uuidCfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Unknown still counts as set. An interpolated UUID must not pick up the
	// production default, and both attributes in config is a conflict even
	// before the values are known. Coolify looks up the name first.
	nameSet := configPresent(nameCfg)
	uuidSet := configPresent(uuidCfg)
	if nameSet && uuidSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("environment_uuid"),
			"Conflicting environment identity",
			"Set environment_name or environment_uuid, not both. Coolify looks up the name first, so a name would ignore the UUID.",
		)
		return
	}
	if uuidSet {
		planUUIDOnlyEnvironment(req, resp)
		return
	}
	if nameCfg.IsUnknown() {
		planUnknownEnvironmentName(req, resp)
		return
	}
	if !nameSet {
		if knownString(req.StateValue) && !req.State.Raw.IsNull() {
			resp.PlanValue = req.StateValue
		} else {
			resp.PlanValue = types.StringValue("production")
		}
	}
	markKnownEnvironmentNameReplace(req, resp)
}

// planUUIDOnlyEnvironment clears the name. A stored name means this resource
// was addressed by name before, and Coolify will not move it in place.
func planUUIDOnlyEnvironment(req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	resp.PlanValue = types.StringNull()
	if knownString(req.StateValue) && !req.State.Raw.IsNull() {
		resp.RequiresReplace = true
	}
}

// planUnknownEnvironmentName keeps an unknown name. Update cannot send
// environment_name, so a known stored name has to be replaced.
func planUnknownEnvironmentName(req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.State.Raw.IsNull() && knownString(req.StateValue) {
		resp.RequiresReplace = true
	}
}

func markKnownEnvironmentNameReplace(req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() {
		return
	}
	planValue := resp.PlanValue
	if planValue.IsNull() || planValue.IsUnknown() {
		planValue = req.PlanValue
	}
	if !knownString(req.StateValue) || !knownString(planValue) {
		return
	}
	if !planValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// configPresent is true when the attribute is in configuration, including an
// unknown interpolation. A known empty string is treated as omitted.
func configPresent(v types.String) bool {
	if v.IsNull() {
		return false
	}
	if v.IsUnknown() {
		return true
	}
	return v.ValueString() != ""
}

func knownString(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown()
}
