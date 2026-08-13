package notificationtelegram

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*telegramResource)(nil)
	_ resource.ResourceWithConfigure   = (*telegramResource)(nil)
	_ resource.ResourceWithImportState = (*telegramResource)(nil)
)

type telegramResource struct {
	client *client.Client
}

type model struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Token   types.String `tfsdk:"token"`
	ChatID  types.String `tfsdk:"chat_id"`

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

	ThreadDeploymentSuccess    types.String `tfsdk:"thread_deployment_success"`
	ThreadDeploymentFailure    types.String `tfsdk:"thread_deployment_failure"`
	ThreadStatusChange         types.String `tfsdk:"thread_status_change"`
	ThreadBackupSuccess        types.String `tfsdk:"thread_backup_success"`
	ThreadBackupFailure        types.String `tfsdk:"thread_backup_failure"`
	ThreadScheduledTaskSuccess types.String `tfsdk:"thread_scheduled_task_success"`
	ThreadScheduledTaskFailure types.String `tfsdk:"thread_scheduled_task_failure"`
	ThreadDockerCleanupSuccess types.String `tfsdk:"thread_docker_cleanup_success"`
	ThreadDockerCleanupFailure types.String `tfsdk:"thread_docker_cleanup_failure"`
	ThreadServerDiskUsage      types.String `tfsdk:"thread_server_disk_usage"`
	ThreadServerReachable      types.String `tfsdk:"thread_server_reachable"`
	ThreadServerUnreachable    types.String `tfsdk:"thread_server_unreachable"`
	ThreadServerPatch          types.String `tfsdk:"thread_server_patch"`
	ThreadTraefikOutdated      types.String `tfsdk:"thread_traefik_outdated"`
}

func NewResource() resource.Resource {
	return &telegramResource{}
}

func (r *telegramResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_telegram"
}

func (r *telegramResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	sensStr := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc + " Sensitive; Coolify may omit it on read unless the API token can read sensitive fields (`read:sensitive` or root). Preserve after import.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		}
	}
	attrs := map[string]schema.Attribute{
		"id":                            notificationcommon.IDAttribute(),
		"enabled":                       notificationcommon.EnabledAttribute("Telegram"),
		"token":                         sensStr("Telegram bot token."),
		"chat_id":                       sensStr("Telegram chat ID."),
		"thread_deployment_success":     sensStr("Telegram forum thread ID for deployment_success events."),
		"thread_deployment_failure":     sensStr("Telegram forum thread ID for deployment_failure events."),
		"thread_status_change":          sensStr("Telegram forum thread ID for status_change events."),
		"thread_backup_success":         sensStr("Telegram forum thread ID for backup_success events."),
		"thread_backup_failure":         sensStr("Telegram forum thread ID for backup_failure events."),
		"thread_scheduled_task_success": sensStr("Telegram forum thread ID for scheduled_task_success events."),
		"thread_scheduled_task_failure": sensStr("Telegram forum thread ID for scheduled_task_failure events."),
		"thread_docker_cleanup_success": sensStr("Telegram forum thread ID for docker_cleanup_success events."),
		"thread_docker_cleanup_failure": sensStr("Telegram forum thread ID for docker_cleanup_failure events."),
		"thread_server_disk_usage":      sensStr("Telegram forum thread ID for server_disk_usage events."),
		"thread_server_reachable":       sensStr("Telegram forum thread ID for server_reachable events."),
		"thread_server_unreachable":     sensStr("Telegram forum thread ID for server_unreachable events."),
		"thread_server_patch":           sensStr("Telegram forum thread ID for server_patch events."),
		"thread_traefik_outdated":       sensStr("Telegram forum thread ID for traefik_outdated events."),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the current team's Telegram notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"**Requires Coolify >= v4.3.0** (notification routes are absent on v4.2.x and older; acceptance tests skip when the API is missing).\n\n" +
			"On destroy, Telegram notifications are disabled (`enabled = false`); token, chat ID, and thread IDs are left unchanged. " +
			"Import with id `current`.",
		Attributes: notificationcommon.MergeAttrs(attrs, notificationcommon.EventSchemaAttrs("Telegram")),
	}
}

func (r *telegramResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *telegramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	updated, err := r.client.UpdateTelegramNotifications(ctx, createInputFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Telegram notifications", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *telegramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	got, err := r.client.GetTelegramNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Telegram notifications", err.Error())
		return
	}
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *telegramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	updated, err := r.client.UpdateTelegramNotifications(ctx, updateInputFromPlan(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Telegram notifications", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *telegramResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling Telegram notifications)", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	enabled := false
	_, err := r.client.UpdateTelegramNotifications(ctx, client.UpdateTelegramNotificationInput{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling Telegram notifications on destroy", err.Error())
	}
}

func (r *telegramResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_notification_telegram")
}

func createInputFromPlan(plan model) client.UpdateTelegramNotificationInput {
	in := client.UpdateTelegramNotificationInput{
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
	if flex.StringValueConfigured(plan.Token) {
		s := plan.Token.ValueString()
		in.Token = &s
	}
	if flex.StringValueConfigured(plan.ChatID) {
		s := plan.ChatID.ValueString()
		in.ChatID = &s
	}
	if flex.StringValueConfigured(plan.ThreadDeploymentSuccess) {
		s := plan.ThreadDeploymentSuccess.ValueString()
		in.ThreadDeploymentSuccess = &s
	}
	if flex.StringValueConfigured(plan.ThreadDeploymentFailure) {
		s := plan.ThreadDeploymentFailure.ValueString()
		in.ThreadDeploymentFailure = &s
	}
	if flex.StringValueConfigured(plan.ThreadStatusChange) {
		s := plan.ThreadStatusChange.ValueString()
		in.ThreadStatusChange = &s
	}
	if flex.StringValueConfigured(plan.ThreadBackupSuccess) {
		s := plan.ThreadBackupSuccess.ValueString()
		in.ThreadBackupSuccess = &s
	}
	if flex.StringValueConfigured(plan.ThreadBackupFailure) {
		s := plan.ThreadBackupFailure.ValueString()
		in.ThreadBackupFailure = &s
	}
	if flex.StringValueConfigured(plan.ThreadScheduledTaskSuccess) {
		s := plan.ThreadScheduledTaskSuccess.ValueString()
		in.ThreadScheduledTaskSuccess = &s
	}
	if flex.StringValueConfigured(plan.ThreadScheduledTaskFailure) {
		s := plan.ThreadScheduledTaskFailure.ValueString()
		in.ThreadScheduledTaskFailure = &s
	}
	if flex.StringValueConfigured(plan.ThreadDockerCleanupSuccess) {
		s := plan.ThreadDockerCleanupSuccess.ValueString()
		in.ThreadDockerCleanupSuccess = &s
	}
	if flex.StringValueConfigured(plan.ThreadDockerCleanupFailure) {
		s := plan.ThreadDockerCleanupFailure.ValueString()
		in.ThreadDockerCleanupFailure = &s
	}
	if flex.StringValueConfigured(plan.ThreadServerDiskUsage) {
		s := plan.ThreadServerDiskUsage.ValueString()
		in.ThreadServerDiskUsage = &s
	}
	if flex.StringValueConfigured(plan.ThreadServerReachable) {
		s := plan.ThreadServerReachable.ValueString()
		in.ThreadServerReachable = &s
	}
	if flex.StringValueConfigured(plan.ThreadServerUnreachable) {
		s := plan.ThreadServerUnreachable.ValueString()
		in.ThreadServerUnreachable = &s
	}
	if flex.StringValueConfigured(plan.ThreadServerPatch) {
		s := plan.ThreadServerPatch.ValueString()
		in.ThreadServerPatch = &s
	}
	if flex.StringValueConfigured(plan.ThreadTraefikOutdated) {
		s := plan.ThreadTraefikOutdated.ValueString()
		in.ThreadTraefikOutdated = &s
	}
	return in
}

func updateInputFromPlan(plan, state model) client.UpdateTelegramNotificationInput {
	in := client.UpdateTelegramNotificationInput{
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
	if w := flex.StringIfChanged(plan.Token, state.Token); w != nil {
		in.Token = w
	}
	if w := flex.StringIfChanged(plan.ChatID, state.ChatID); w != nil {
		in.ChatID = w
	}
	if w := flex.StringIfChanged(plan.ThreadDeploymentSuccess, state.ThreadDeploymentSuccess); w != nil {
		in.ThreadDeploymentSuccess = w
	}
	if w := flex.StringIfChanged(plan.ThreadDeploymentFailure, state.ThreadDeploymentFailure); w != nil {
		in.ThreadDeploymentFailure = w
	}
	if w := flex.StringIfChanged(plan.ThreadStatusChange, state.ThreadStatusChange); w != nil {
		in.ThreadStatusChange = w
	}
	if w := flex.StringIfChanged(plan.ThreadBackupSuccess, state.ThreadBackupSuccess); w != nil {
		in.ThreadBackupSuccess = w
	}
	if w := flex.StringIfChanged(plan.ThreadBackupFailure, state.ThreadBackupFailure); w != nil {
		in.ThreadBackupFailure = w
	}
	if w := flex.StringIfChanged(plan.ThreadScheduledTaskSuccess, state.ThreadScheduledTaskSuccess); w != nil {
		in.ThreadScheduledTaskSuccess = w
	}
	if w := flex.StringIfChanged(plan.ThreadScheduledTaskFailure, state.ThreadScheduledTaskFailure); w != nil {
		in.ThreadScheduledTaskFailure = w
	}
	if w := flex.StringIfChanged(plan.ThreadDockerCleanupSuccess, state.ThreadDockerCleanupSuccess); w != nil {
		in.ThreadDockerCleanupSuccess = w
	}
	if w := flex.StringIfChanged(plan.ThreadDockerCleanupFailure, state.ThreadDockerCleanupFailure); w != nil {
		in.ThreadDockerCleanupFailure = w
	}
	if w := flex.StringIfChanged(plan.ThreadServerDiskUsage, state.ThreadServerDiskUsage); w != nil {
		in.ThreadServerDiskUsage = w
	}
	if w := flex.StringIfChanged(plan.ThreadServerReachable, state.ThreadServerReachable); w != nil {
		in.ThreadServerReachable = w
	}
	if w := flex.StringIfChanged(plan.ThreadServerUnreachable, state.ThreadServerUnreachable); w != nil {
		in.ThreadServerUnreachable = w
	}
	if w := flex.StringIfChanged(plan.ThreadServerPatch, state.ThreadServerPatch); w != nil {
		in.ThreadServerPatch = w
	}
	if w := flex.StringIfChanged(plan.ThreadTraefikOutdated, state.ThreadTraefikOutdated); w != nil {
		in.ThreadTraefikOutdated = w
	}
	return in
}

func flatten(api *client.TelegramNotificationSettings, m *model) {
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.Enabled = types.BoolValue(api.Enabled)
	flex.SetStringPreserveEmpty(&m.Token, api.Token)
	flex.SetStringPreserveEmpty(&m.ChatID, api.ChatID)
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
	flex.SetStringPreserveEmpty(&m.ThreadDeploymentSuccess, api.ThreadDeploymentSuccess)
	flex.SetStringPreserveEmpty(&m.ThreadDeploymentFailure, api.ThreadDeploymentFailure)
	flex.SetStringPreserveEmpty(&m.ThreadStatusChange, api.ThreadStatusChange)
	flex.SetStringPreserveEmpty(&m.ThreadBackupSuccess, api.ThreadBackupSuccess)
	flex.SetStringPreserveEmpty(&m.ThreadBackupFailure, api.ThreadBackupFailure)
	flex.SetStringPreserveEmpty(&m.ThreadScheduledTaskSuccess, api.ThreadScheduledTaskSuccess)
	flex.SetStringPreserveEmpty(&m.ThreadScheduledTaskFailure, api.ThreadScheduledTaskFailure)
	flex.SetStringPreserveEmpty(&m.ThreadDockerCleanupSuccess, api.ThreadDockerCleanupSuccess)
	flex.SetStringPreserveEmpty(&m.ThreadDockerCleanupFailure, api.ThreadDockerCleanupFailure)
	flex.SetStringPreserveEmpty(&m.ThreadServerDiskUsage, api.ThreadServerDiskUsage)
	flex.SetStringPreserveEmpty(&m.ThreadServerReachable, api.ThreadServerReachable)
	flex.SetStringPreserveEmpty(&m.ThreadServerUnreachable, api.ThreadServerUnreachable)
	flex.SetStringPreserveEmpty(&m.ThreadServerPatch, api.ThreadServerPatch)
	flex.SetStringPreserveEmpty(&m.ThreadTraefikOutdated, api.ThreadTraefikOutdated)
}
