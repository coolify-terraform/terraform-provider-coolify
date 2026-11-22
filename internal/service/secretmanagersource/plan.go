package secretmanagersource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// keepProviderUntilTokenChanges copies the stored provider into the plan
// while integration_token_uuid is unchanged. UseStateForUnknown would keep
// the old provider after the token changes, and apply would then disagree
// with the plan. The attribute cannot be named provider: Terraform treats
// that name as a meta-argument.
type keepProviderUntilTokenChanges struct{}

func (keepProviderUntilTokenChanges) Description(context.Context) string {
	return "Keeps the stored integration provider unless integration_token_uuid changes."
}

func (m keepProviderUntilTokenChanges) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (keepProviderUntilTokenChanges) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var planToken, stateToken types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("integration_token_uuid"), &planToken)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("integration_token_uuid"), &stateToken)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tokenSame := !planToken.IsNull() && !planToken.IsUnknown() && planToken.Equal(stateToken)
	if tokenSame && !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		return
	}
	if !tokenSame && !req.State.Raw.IsNull() {
		resp.PlanValue = types.StringUnknown()
	}
}
