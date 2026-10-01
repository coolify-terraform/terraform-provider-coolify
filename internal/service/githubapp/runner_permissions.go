package githubapp

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const runnerPermissionDescription = "Read-only GitHub App runner permission stored by Coolify 4.4 tip. " +
	"The public create and update APIs do not accept this field. Empty on older Coolify versions."

func githubRunnerPermissionResourceAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"actions": schema.StringAttribute{
			MarkdownDescription: "Repository Actions permission (`read` or `write`). " + runnerPermissionDescription,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"organization_self_hosted_runners": schema.StringAttribute{
			MarkdownDescription: "Organization self-hosted runners permission. " + runnerPermissionDescription,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"webhook_events": schema.ListAttribute{
			MarkdownDescription: "Webhook events configured on the GitHub App. " + runnerPermissionDescription,
			ElementType:         types.StringType,
			Computed:            true,
			PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
		},
		"runner_group_id": schema.Int64Attribute{
			MarkdownDescription: "GitHub Actions runner group id. " + runnerPermissionDescription,
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
	}
}

func applyGitHubRunnerPermissions(app *client.GitHubApp, actions, org *types.String, events *types.List, group *types.Int64) {
	*actions = flex.StringToFramework(app.Actions)
	*org = flex.StringToFramework(app.OrganizationSelfHostedRunners)
	if app.RunnerGroupID != nil {
		*group = types.Int64Value(*app.RunnerGroupID)
	} else {
		*group = types.Int64Null()
	}
	if app.WebhookEvents == nil {
		*events = types.ListNull(types.StringType)
		return
	}
	elems := make([]attr.Value, len(app.WebhookEvents))
	for i, ev := range app.WebhookEvents {
		elems[i] = types.StringValue(ev)
	}
	*events = types.ListValueMust(types.StringType, elems)
}
