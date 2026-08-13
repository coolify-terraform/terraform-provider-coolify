package notificationwebhook

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const importIDCurrent = "current"

var (
	_ resource.Resource                = (*webhookResource)(nil)
	_ resource.ResourceWithConfigure   = (*webhookResource)(nil)
	_ resource.ResourceWithImportState = (*webhookResource)(nil)
)

type webhookResource struct {
	client *client.Client
}

type model struct {
	ID         types.String `tfsdk:"id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	WebhookURL types.String `tfsdk:"webhook_url"`

	DeploymentSuccess    types.Bool `tfsdk:"deployment_success"`
	DeploymentFailure    types.Bool `tfsdk:"deployment_failure"`
	StatusChange         types.Bool `tfsdk:"status_change"`
	BackupSuccess        types.Bool `tfsdk:"backup_success"`
	BackupFailure        types.Bool `tfsdk:"backup_failure"`
	ScheduledTaskSuccess types.Bool `tfsdk:"scheduled_task_success"`
	ScheduledTaskFailure types.Bool `tfsdk:"scheduled_task_failure"`
	DockerCleanupSuccess types.Bool `tfsdk:"docker_cleanup_success"`
	DockerCleanupFailure types.Bool `tfsdk:"docker_cleanup_failure"`
	ServerDiskUsage      types.Bool `tfsdk:"server_disk_usage"`
	ServerReachable      types.Bool `tfsdk:"server_reachable"`
	ServerUnreachable    types.Bool `tfsdk:"server_unreachable"`
	ServerPatch          types.Bool `tfsdk:"server_patch"`
	TraefikOutdated      types.Bool `tfsdk:"traefik_outdated"`
}

// NewResource returns a new Webhook notification resource.
func NewResource() resource.Resource {
	return &webhookResource{}
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	eventDesc := func(event string) string {
		return fmt.Sprintf("Whether to send Webhook notifications for %s events.", event)
	}
	boolOptComputed := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{
			MarkdownDescription: desc,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the current team's Webhook notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"Requires Coolify >= v4.3.0.\n\n" +
			"On destroy, Webhook notifications are disabled (`enabled = false`); the webhook URL is left unchanged. " +
			"Import with id `current`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier. Always `current` (team is implied by the API token).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Webhook notifications are enabled for the team.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"webhook_url": schema.StringAttribute{
				MarkdownDescription: "Webhook incoming webhook URL. Sensitive; Coolify may omit it on read unless the API token " +
					"can read sensitive fields (`read:sensitive` or root). Preserve the value in configuration after import. " +
					"Must pass Coolify's SafeWebhookUrl rule (public http/https; private/loopback hosts rejected).",
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_success":     boolOptComputed(eventDesc("deployment success")),
			"deployment_failure":     boolOptComputed(eventDesc("deployment failure")),
			"status_change":          boolOptComputed(eventDesc("status change")),
			"backup_success":         boolOptComputed(eventDesc("backup success")),
			"backup_failure":         boolOptComputed(eventDesc("backup failure")),
			"scheduled_task_success": boolOptComputed(eventDesc("scheduled task success")),
			"scheduled_task_failure": boolOptComputed(eventDesc("scheduled task failure")),
			"docker_cleanup_success": boolOptComputed(eventDesc("Docker cleanup success")),
			"docker_cleanup_failure": boolOptComputed(eventDesc("Docker cleanup failure")),
			"server_disk_usage":      boolOptComputed(eventDesc("server disk usage")),
			"server_reachable":       boolOptComputed(eventDesc("server reachable")),
			"server_unreachable":     boolOptComputed(eventDesc("server unreachable")),
			"server_patch":           boolOptComputed(eventDesc("server patch")),
			"traefik_outdated":       boolOptComputed(eventDesc("Traefik outdated")),
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_webhook"})

	updated, err := r.client.UpdateWebhookNotifications(ctx, createInputFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Webhook notifications", err.Error())
		return
	}

	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_webhook"})

	got, err := r.client.GetWebhookNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Webhook notifications", err.Error())
		return
	}

	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_webhook"})

	updated, err := r.client.UpdateWebhookNotifications(ctx, updateInputFromPlan(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Webhook notifications", err.Error())
		return
	}

	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling Webhook notifications)", map[string]interface{}{"resource_type": "coolify_notification_webhook"})
	enabled := false
	_, err := r.client.UpdateWebhookNotifications(ctx, client.UpdateWebhookNotificationInput{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling Webhook notifications on destroy", err.Error())
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id != "" && id != importIDCurrent {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("coolify_notification_webhook is a team singleton; import with id %q (got %q)", importIDCurrent, id),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importIDCurrent)...)
}

func createInputFromPlan(plan model) client.UpdateWebhookNotificationInput {
	in := client.UpdateWebhookNotificationInput{
		Enabled:              flex.BoolValueOrNull(plan.Enabled),
		DeploymentSuccess:    flex.BoolValueOrNull(plan.DeploymentSuccess),
		DeploymentFailure:    flex.BoolValueOrNull(plan.DeploymentFailure),
		StatusChange:         flex.BoolValueOrNull(plan.StatusChange),
		BackupSuccess:        flex.BoolValueOrNull(plan.BackupSuccess),
		BackupFailure:        flex.BoolValueOrNull(plan.BackupFailure),
		ScheduledTaskSuccess: flex.BoolValueOrNull(plan.ScheduledTaskSuccess),
		ScheduledTaskFailure: flex.BoolValueOrNull(plan.ScheduledTaskFailure),
		DockerCleanupSuccess: flex.BoolValueOrNull(plan.DockerCleanupSuccess),
		DockerCleanupFailure: flex.BoolValueOrNull(plan.DockerCleanupFailure),
		ServerDiskUsage:      flex.BoolValueOrNull(plan.ServerDiskUsage),
		ServerReachable:      flex.BoolValueOrNull(plan.ServerReachable),
		ServerUnreachable:    flex.BoolValueOrNull(plan.ServerUnreachable),
		ServerPatch:          flex.BoolValueOrNull(plan.ServerPatch),
		TraefikOutdated:      flex.BoolValueOrNull(plan.TraefikOutdated),
	}
	if flex.StringValueConfigured(plan.WebhookURL) {
		v := plan.WebhookURL.ValueString()
		in.Webhook = &v
	}
	return in
}

func updateInputFromPlan(plan, state model) client.UpdateWebhookNotificationInput {
	in := client.UpdateWebhookNotificationInput{
		Enabled:              flex.BoolIfChanged(plan.Enabled, state.Enabled),
		DeploymentSuccess:    flex.BoolIfChanged(plan.DeploymentSuccess, state.DeploymentSuccess),
		DeploymentFailure:    flex.BoolIfChanged(plan.DeploymentFailure, state.DeploymentFailure),
		StatusChange:         flex.BoolIfChanged(plan.StatusChange, state.StatusChange),
		BackupSuccess:        flex.BoolIfChanged(plan.BackupSuccess, state.BackupSuccess),
		BackupFailure:        flex.BoolIfChanged(plan.BackupFailure, state.BackupFailure),
		ScheduledTaskSuccess: flex.BoolIfChanged(plan.ScheduledTaskSuccess, state.ScheduledTaskSuccess),
		ScheduledTaskFailure: flex.BoolIfChanged(plan.ScheduledTaskFailure, state.ScheduledTaskFailure),
		DockerCleanupSuccess: flex.BoolIfChanged(plan.DockerCleanupSuccess, state.DockerCleanupSuccess),
		DockerCleanupFailure: flex.BoolIfChanged(plan.DockerCleanupFailure, state.DockerCleanupFailure),
		ServerDiskUsage:      flex.BoolIfChanged(plan.ServerDiskUsage, state.ServerDiskUsage),
		ServerReachable:      flex.BoolIfChanged(plan.ServerReachable, state.ServerReachable),
		ServerUnreachable:    flex.BoolIfChanged(plan.ServerUnreachable, state.ServerUnreachable),
		ServerPatch:          flex.BoolIfChanged(plan.ServerPatch, state.ServerPatch),
		TraefikOutdated:      flex.BoolIfChanged(plan.TraefikOutdated, state.TraefikOutdated),
	}
	if w := flex.StringIfChanged(plan.WebhookURL, state.WebhookURL); w != nil {
		in.Webhook = w
	}
	return in
}

func flatten(api *client.WebhookNotificationSettings, m *model) {
	m.ID = types.StringValue(importIDCurrent)
	m.Enabled = types.BoolValue(api.Enabled)
	flex.SetStringPreserveEmpty(&m.WebhookURL, api.Webhook)
	m.DeploymentSuccess = types.BoolValue(api.DeploymentSuccess)
	m.DeploymentFailure = types.BoolValue(api.DeploymentFailure)
	m.StatusChange = types.BoolValue(api.StatusChange)
	m.BackupSuccess = types.BoolValue(api.BackupSuccess)
	m.BackupFailure = types.BoolValue(api.BackupFailure)
	m.ScheduledTaskSuccess = types.BoolValue(api.ScheduledTaskSuccess)
	m.ScheduledTaskFailure = types.BoolValue(api.ScheduledTaskFailure)
	m.DockerCleanupSuccess = types.BoolValue(api.DockerCleanupSuccess)
	m.DockerCleanupFailure = types.BoolValue(api.DockerCleanupFailure)
	m.ServerDiskUsage = types.BoolValue(api.ServerDiskUsage)
	m.ServerReachable = types.BoolValue(api.ServerReachable)
	m.ServerUnreachable = types.BoolValue(api.ServerUnreachable)
	m.ServerPatch = types.BoolValue(api.ServerPatch)
	m.TraefikOutdated = types.BoolValue(api.TraefikOutdated)
}
