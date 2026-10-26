package hostinger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/server"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &hostingerServerResource{}
	_ resource.ResourceWithConfigure   = &hostingerServerResource{}
	_ resource.ResourceWithImportState = &hostingerServerResource{}
	_ resource.ResourceWithModifyPlan  = &hostingerServerResource{}
)

type hostingerServerResource struct {
	client *client.Client
}

type hostingerServerResourceModel struct {
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
	UUID        types.String   `tfsdk:"uuid"`
	IP          types.String   `tfsdk:"ip"`
	IsReachable types.Bool     `tfsdk:"is_reachable"`
	IsUsable    types.Bool     `tfsdk:"is_usable"`

	CloudProviderTokenUUID    types.String `tfsdk:"cloud_provider_token_uuid"`
	ItemID                    types.String `tfsdk:"item_id"`
	DataCenterID              types.Int64  `tfsdk:"data_center_id"`
	TemplateID                types.Int64  `tfsdk:"template_id"`
	PublicKeyIDs              types.List   `tfsdk:"public_key_ids"`
	PostInstallScriptID       types.Int64  `tfsdk:"post_install_script_id"`
	EnableBackups             types.Bool   `tfsdk:"enable_backups"`
	InstantValidate           types.Bool   `tfsdk:"instant_validate"`
	HostingerVirtualMachineID types.Int64  `tfsdk:"hostinger_virtual_machine_id"`
	HostingerVMStatus         types.String `tfsdk:"hostinger_virtual_machine_status"`
	Provisioning              types.Bool   `tfsdk:"provisioning"`

	Name                                     types.String `tfsdk:"name"`
	Description                              types.String `tfsdk:"description"`
	Port                                     types.Int64  `tfsdk:"port"`
	User                                     types.String `tfsdk:"user"`
	PrivateKeyUUID                           types.String `tfsdk:"private_key_uuid"`
	IsBuildServer                            types.Bool   `tfsdk:"is_build_server"`
	ServerRole                               types.String `tfsdk:"server_role"`
	ConcurrentBuilds                         types.Int64  `tfsdk:"concurrent_builds"`
	DynamicTimeout                           types.Int64  `tfsdk:"dynamic_timeout"`
	DeploymentQueueLimit                     types.Int64  `tfsdk:"deployment_queue_limit"`
	ConnectionTimeout                        types.Int64  `tfsdk:"connection_timeout"`
	ServerDiskUsageNotificationThreshold     types.Int64  `tfsdk:"server_disk_usage_notification_threshold"`
	ServerDiskUsageNotificationIntervalHours types.Int64  `tfsdk:"server_disk_usage_notification_interval_hours"`
	ServerDiskUsageCheckFrequency            types.String `tfsdk:"server_disk_usage_check_frequency"`
	WildcardDomain                           types.String `tfsdk:"wildcard_domain"`
	IsCloudFlareTunnel                       types.Bool   `tfsdk:"is_cloudflare_tunnel"`
	ServerTimezone                           types.String `tfsdk:"server_timezone"`
	IsMetricsEnabled                         types.Bool   `tfsdk:"is_metrics_enabled"`
	IsTerminalEnabled                        types.Bool   `tfsdk:"is_terminal_enabled"`
	IsSentinelEnabled                        types.Bool   `tfsdk:"is_sentinel_enabled"`
	SentinelMetricsHistoryDays               types.Int64  `tfsdk:"sentinel_metrics_history_days"`
	SentinelMetricsRefreshRateSeconds        types.Int64  `tfsdk:"sentinel_metrics_refresh_rate_seconds"`
	SentinelPushIntervalSeconds              types.Int64  `tfsdk:"sentinel_push_interval_seconds"`
	DockerCleanupFrequency                   types.String `tfsdk:"docker_cleanup_frequency"`
	DockerCleanupThreshold                   types.Int64  `tfsdk:"docker_cleanup_threshold"`
	ForceDockerCleanup                       types.Bool   `tfsdk:"force_docker_cleanup"`
	DeleteUnusedVolumes                      types.Bool   `tfsdk:"delete_unused_volumes"`
	DeleteUnusedNetworks                     types.Bool   `tfsdk:"delete_unused_networks"`
	GenerateExactLabels                      types.Bool   `tfsdk:"generate_exact_labels"`
	ForceDisabled                            types.Bool   `tfsdk:"force_disabled"`
	IsJumpServer                             types.Bool   `tfsdk:"is_jump_server"`
	IsSwarmManager                           types.Bool   `tfsdk:"is_swarm_manager"`
	IsSwarmWorker                            types.Bool   `tfsdk:"is_swarm_worker"`
	IsSentinelDebugEnabled                   types.Bool   `tfsdk:"is_sentinel_debug_enabled"`
	DisableApplicationImageRetention         types.Bool   `tfsdk:"disable_application_image_retention"`
	SentinelToken                            types.String `tfsdk:"sentinel_token"`
	SentinelCustomURL                        types.String `tfsdk:"sentinel_custom_url"`
	IsLogdrainAxiomEnabled                   types.Bool   `tfsdk:"is_logdrain_axiom_enabled"`
	LogdrainAxiomAPIKey                      types.String `tfsdk:"logdrain_axiom_api_key"`
	LogdrainAxiomDatasetName                 types.String `tfsdk:"logdrain_axiom_dataset_name"`
	IsLogdrainCustomEnabled                  types.Bool   `tfsdk:"is_logdrain_custom_enabled"`
	LogdrainCustomConfig                     types.String `tfsdk:"logdrain_custom_config"`
	LogdrainCustomConfigParser               types.String `tfsdk:"logdrain_custom_config_parser"`
	IsLogdrainHighlightEnabled               types.Bool   `tfsdk:"is_logdrain_highlight_enabled"`
	LogdrainHighlightProjectID               types.String `tfsdk:"logdrain_highlight_project_id"`
	IsLogdrainNewrelicEnabled                types.Bool   `tfsdk:"is_logdrain_newrelic_enabled"`
	LogdrainNewrelicBaseURI                  types.String `tfsdk:"logdrain_newrelic_base_uri"`
	LogdrainNewrelicLicenseKey               types.String `tfsdk:"logdrain_newrelic_license_key"`
	ComposeVersion                           types.String `tfsdk:"compose_version"`
	ComposeVersionCheckedAt                  types.String `tfsdk:"compose_version_checked_at"`
	DockerVersion                            types.String `tfsdk:"docker_version"`
	DockerVersionCheckedAt                   types.String `tfsdk:"docker_version_checked_at"`
	BackupCompressionCPUPercentage           types.Int64  `tfsdk:"backup_compression_cpu_percentage"`
}

func NewResource() resource.Resource { return &hostingerServerResource{} }

func (r *hostingerServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_hostinger"
}

func (r *hostingerServerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Purchases a Hostinger VPS and registers it with Coolify. Requires Coolify >= v4.5.0.\n\n" +
			"~> **Warning:** Deleting this resource deletes the server from Coolify and cascade-deletes applications, databases, and services on it. Coolify does not cancel the Hostinger VPS. The subscription keeps billing in hPanel.\n\n" +
			"~> **Import note:** Purchase fields (`cloud_provider_token_uuid`, `item_id`, `data_center_id`, `template_id`, `public_key_ids`, `post_install_script_id`, `enable_backups`) are only sent at creation and are not returned by the server GET. After import they are empty. Set them in configuration to the values used at purchase. Terraform adopts that configuration and does not buy another VPS. Changing a value that is already in state replaces the server and purchases again. Destroy still does not cancel the previous subscription.",
		Attributes: server.CommonServerAttrs(ctx, hostingerSchemaAttributes()),
	}
}

func hostingerSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Name stored on the Coolify server. Coolify lowercases it. Hostinger uses that name as the VPS hostname only when it contains a dot, such as `app.example.com`. A single label such as `app-node` stays on the Coolify server, and Hostinger chooses the machine hostname.",
			Required:            true,
		},
		"ip": schema.StringAttribute{
			MarkdownDescription: "Public IP Coolify stored for this server. `1.2.3.4` means Hostinger has not published an address yet.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"cloud_provider_token_uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of a `coolify_cloud_token` whose provider is `hostinger`. Changing a known value forces a new server. A value missing after import is adopted and does not purchase another VPS.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{flex.RequiresReplaceIfKnown()},
			Validators:          []validator.String{validate.UUID()},
		},
		"item_id": schema.StringAttribute{
			MarkdownDescription: "Hostinger catalog item id, for example `hostingercom-vps-kvm2`. Use `coolify_hostinger_catalog`. Changing a known value forces a new server. A value missing after import is adopted and does not purchase another VPS.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{flex.RequiresReplaceIfKnown()},
		},
		"data_center_id": schema.Int64Attribute{
			MarkdownDescription: "Hostinger data center id. Use `coolify_hostinger_data_centers`. Changing a known value forces a new server. A value missing after import is adopted and does not purchase another VPS.",
			Required:            true,
			PlanModifiers:       []planmodifier.Int64{flex.Int64RequiresReplaceIfKnown()},
		},
		"template_id": schema.Int64Attribute{
			MarkdownDescription: "Hostinger OS template id. Use `coolify_hostinger_templates`. Changing a known value forces a new server. A value missing after import is adopted and does not purchase another VPS.",
			Required:            true,
			PlanModifiers:       []planmodifier.Int64{flex.Int64RequiresReplaceIfKnown()},
		},
		"public_key_ids": schema.ListAttribute{
			MarkdownDescription: "Extra Hostinger SSH public key ids to attach at purchase. Requires a Coolify token with the deploy ability. Changing a known value forces a new server. A value missing after import is adopted and does not purchase another VPS.",
			Optional:            true,
			ElementType:         types.Int64Type,
			PlanModifiers:       []planmodifier.List{flex.ListRequiresReplaceIfKnown()},
		},
		"post_install_script_id": schema.Int64Attribute{
			MarkdownDescription: "Hostinger post-install script id. Requires a Coolify token with the deploy ability. Changing a known value forces a new server. A value missing after import is adopted and does not purchase another VPS.",
			Optional:            true,
			PlanModifiers:       []planmodifier.Int64{flex.Int64RequiresReplaceIfKnown()},
		},
		"enable_backups": schema.BoolAttribute{
			MarkdownDescription: "Ask Hostinger to enable VPS backups at purchase. Omitted means false. The API does not return this flag. Changing a known value forces a new server.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				flex.BoolRequiresReplaceIfKnown(),
			},
		},
		"instant_validate": schema.BoolAttribute{
			MarkdownDescription: "Validate server connectivity after creation or when this value changes. Defaults to false. Coolify only dispatches validation when the public IP is already known.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"hostinger_virtual_machine_id": schema.Int64Attribute{
			MarkdownDescription: "Hostinger virtual machine id returned by the purchase.",
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"hostinger_virtual_machine_status": schema.StringAttribute{
			MarkdownDescription: "Hostinger virtual machine state from the server read, when Coolify includes it.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"provisioning": schema.BoolAttribute{
			MarkdownDescription: "True while the server IP is still a placeholder (`1.2.3.4`, `0.0.0.0`, or `::`).",
			Computed:            true,
		},
	}
}

func (r *hostingerServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *hostingerServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostingerServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client != nil && !r.client.SupportsCoolify45() {
		ver := r.client.CoolifyVersion
		if ver == "" {
			ver = "unknown"
		}
		resp.Diagnostics.AddError(
			"Coolify version cannot purchase a Hostinger server",
			fmt.Sprintf("This Coolify instance (%s) is older than v4.5.0, which added POST /api/v1/servers/hostinger.", ver),
		)
		return
	}
	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	plan.EnableBackups = flex.BoolIfNull(plan.EnableBackups, false)
	keys, keyDiags := int64List(ctx, plan.PublicKeyIDs)
	resp.Diagnostics.Append(keyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := client.CreateHostingerServerInput{
		CloudProviderTokenUUID: plan.CloudProviderTokenUUID.ValueString(),
		ItemID:                 plan.ItemID.ValueString(),
		DataCenterID:           plan.DataCenterID.ValueInt64(),
		TemplateID:             plan.TemplateID.ValueInt64(),
		Name:                   plan.Name.ValueString(),
		PrivateKeyUUID:         plan.PrivateKeyUUID.ValueString(),
		EnableBackups:          flex.BoolValueOrNull(plan.EnableBackups),
		PublicKeyIDs:           keys,
		InstantValidate:        flex.BoolValueOrNull(plan.InstantValidate),
	}
	if !plan.PostInstallScriptID.IsNull() && !plan.PostInstallScriptID.IsUnknown() {
		id := plan.PostInstallScriptID.ValueInt64()
		input.PostInstallScriptID = &id
	}
	created, err := r.client.CreateHostingerServer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Hostinger server", fmt.Sprintf("server %q: %s", plan.Name.ValueString(), err))
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	plan.IP = types.StringValue(created.IP)
	plan.HostingerVirtualMachineID = types.Int64Value(created.HostingerVirtualMachineID)
	plan.Provisioning = types.BoolValue(created.Provisioning)
	server.NormalizeUnknownCloudServerPlanFields(plan.commonPtrs())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := server.ApplyPostCreateCloudProviderSettings(ctx, r.client, created.UUID, plan.commonPtrs(), &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error setting Hostinger server settings", err.Error())
		return
	}
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Hostinger server created but refresh failed",
			fmt.Sprintf("Coolify created Hostinger server %s, but the provider could not read it back: %s. %s", created.UUID, err, flex.CreateReadBackTaintedReplace),
		)
		return
	}
	flattenHostingerServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_server_hostinger", "uuid": created.UUID})
}

func (r *hostingerServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hostingerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Hostinger server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenHostingerServer(srv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostingerServerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan hostingerServerResourceModel
	var cfg hostingerServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var stateBuild *types.Bool
	if !req.State.Raw.IsNull() {
		var state hostingerServerResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateBuild = &state.IsBuildServer
	}
	server.AlignUnconfiguredServerRole(r.client, &cfg.ServerRole, &plan.ServerRole, &plan.IsBuildServer, stateBuild)
	warnHostingerHostname(plan.Name, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// warnHostingerHostname reports that Coolify drops a single-label name
// before the Hostinger purchase. HostingerService::purchaseVirtualMachine
// unsets setup.hostname when it does not contain a dot, then Hostinger
// picks a hostname. The Coolify server name is still saved.
func warnHostingerHostname(name types.String, diags *diag.Diagnostics) {
	if diags == nil || name.IsNull() || name.IsUnknown() || strings.Contains(name.ValueString(), ".") {
		return
	}
	diags.AddWarning(
		"Hostinger will choose the VPS hostname",
		fmt.Sprintf("Coolify only sends %q to Hostinger as the VPS hostname when it contains a dot. The Coolify server keeps that name. Use a name such as app.example.com when the machine hostname should match.", name.ValueString()),
	)
}

func (r *hostingerServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostingerServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state hostingerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	server.WarnDiskUsageInterval(r.client, plan.ServerDiskUsageNotificationIntervalHours, &resp.Diagnostics)
	input := server.BuildServerUpdateInput(plan.commonPtrs(), state.commonPtrs())
	if _, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating Hostinger server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Hostinger server after update", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenHostingerServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostingerServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hostingerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting Hostinger server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
	}
}

func (r *hostingerServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func (m *hostingerServerResourceModel) commonPtrs() server.ServerCommonPtrs {
	return server.ServerCommonPtrs{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		IP: &m.IP, User: &m.User, PrivateKeyUUID: &m.PrivateKeyUUID,
		Port: &m.Port, ConcurrentBuilds: &m.ConcurrentBuilds, DynamicTimeout: &m.DynamicTimeout,
		DeploymentQueueLimit:                     &m.DeploymentQueueLimit,
		ConnectionTimeout:                        &m.ConnectionTimeout,
		ServerDiskUsageNotificationThreshold:     &m.ServerDiskUsageNotificationThreshold,
		ServerDiskUsageNotificationIntervalHours: &m.ServerDiskUsageNotificationIntervalHours,
		ServerDiskUsageCheckFrequency:            &m.ServerDiskUsageCheckFrequency,
		IsBuildServer:                            &m.IsBuildServer, ServerRole: &m.ServerRole, IsReachable: &m.IsReachable, IsUsable: &m.IsUsable,
		InstantValidate: &m.InstantValidate,
		WildcardDomain:  &m.WildcardDomain, IsCloudFlareTunnel: &m.IsCloudFlareTunnel,
		ServerTimezone: &m.ServerTimezone, IsMetricsEnabled: &m.IsMetricsEnabled,
		IsTerminalEnabled: &m.IsTerminalEnabled, IsSentinelEnabled: &m.IsSentinelEnabled,
		SentinelMetricsHistoryDays: &m.SentinelMetricsHistoryDays, SentinelMetricsRefreshRateSeconds: &m.SentinelMetricsRefreshRateSeconds,
		SentinelPushIntervalSeconds: &m.SentinelPushIntervalSeconds,
		DockerCleanupFrequency:      &m.DockerCleanupFrequency, DockerCleanupThreshold: &m.DockerCleanupThreshold,
		ForceDockerCleanup: &m.ForceDockerCleanup, DeleteUnusedVolumes: &m.DeleteUnusedVolumes,
		DeleteUnusedNetworks: &m.DeleteUnusedNetworks, GenerateExactLabels: &m.GenerateExactLabels,
		ForceDisabled: &m.ForceDisabled, IsJumpServer: &m.IsJumpServer,
		IsSwarmManager: &m.IsSwarmManager, IsSwarmWorker: &m.IsSwarmWorker,
		IsSentinelDebugEnabled: &m.IsSentinelDebugEnabled, DisableApplicationImageRetention: &m.DisableApplicationImageRetention,
		SentinelToken: &m.SentinelToken, SentinelCustomURL: &m.SentinelCustomURL,
		IsLogdrainAxiomEnabled: &m.IsLogdrainAxiomEnabled, LogdrainAxiomAPIKey: &m.LogdrainAxiomAPIKey,
		LogdrainAxiomDatasetName: &m.LogdrainAxiomDatasetName, IsLogdrainCustomEnabled: &m.IsLogdrainCustomEnabled,
		LogdrainCustomConfig: &m.LogdrainCustomConfig, LogdrainCustomConfigParser: &m.LogdrainCustomConfigParser,
		IsLogdrainHighlightEnabled: &m.IsLogdrainHighlightEnabled, LogdrainHighlightProjectID: &m.LogdrainHighlightProjectID,
		IsLogdrainNewrelicEnabled: &m.IsLogdrainNewrelicEnabled, LogdrainNewrelicBaseURI: &m.LogdrainNewrelicBaseURI,
		LogdrainNewrelicLicenseKey: &m.LogdrainNewrelicLicenseKey,
		ComposeVersion:             &m.ComposeVersion, ComposeVersionCheckedAt: &m.ComposeVersionCheckedAt,
		DockerVersion: &m.DockerVersion, DockerVersionCheckedAt: &m.DockerVersionCheckedAt,
		BackupCompressionCPUPercentage: &m.BackupCompressionCPUPercentage,
	}
}

func flattenHostingerServer(srv *client.Server, model *hostingerServerResourceModel) {
	priorName := model.Name
	server.FlattenServerCommon(srv, model.commonPtrs())
	if !priorName.IsNull() && !priorName.IsUnknown() && strings.EqualFold(priorName.ValueString(), model.Name.ValueString()) {
		model.Name = priorName
	}
	if srv.HostingerVirtualMachineID != nil {
		model.HostingerVirtualMachineID = types.Int64Value(*srv.HostingerVirtualMachineID)
	} else if model.HostingerVirtualMachineID.IsUnknown() {
		model.HostingerVirtualMachineID = types.Int64Null()
	}
	if srv.HostingerVirtualMachineStatus != "" {
		model.HostingerVMStatus = types.StringValue(srv.HostingerVirtualMachineStatus)
	} else if model.HostingerVMStatus.IsUnknown() {
		model.HostingerVMStatus = types.StringNull()
	}
	model.Provisioning = types.BoolValue(ipIsPlaceholder(model.IP.ValueString()))
}

func ipIsPlaceholder(ip string) bool {
	switch ip {
	case "", "1.2.3.4", "0.0.0.0", "::":
		return true
	default:
		return false
	}
}

func int64List(ctx context.Context, list types.List) ([]int64, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var out []int64
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}
