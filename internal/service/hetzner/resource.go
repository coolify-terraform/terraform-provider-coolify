package hetzner

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &hetznerServerResource{}
	_ resource.ResourceWithConfigure   = &hetznerServerResource{}
	_ resource.ResourceWithImportState = &hetznerServerResource{}
)

type hetznerServerResource struct {
	client *client.Client
}

type hetznerServerResourceModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`
	// Computed after create.
	UUID        types.String `tfsdk:"uuid"`
	IP          types.String `tfsdk:"ip"`
	IsReachable types.Bool   `tfsdk:"is_reachable"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`

	// Hetzner create-only fields (RequiresReplace).
	CloudProviderTokenUUID types.String `tfsdk:"cloud_provider_token_uuid"`
	ServerType             types.String `tfsdk:"server_type"`
	Location               types.String `tfsdk:"location"`
	Image                  types.String `tfsdk:"image"`
	HetznerSSHKeyIDs       types.String `tfsdk:"hetzner_ssh_key_ids"`
	CloudInitScript        types.String `tfsdk:"cloud_init_script"`
	InstantValidate        types.Bool   `tfsdk:"instant_validate"`
	EnableIPv4             types.Bool   `tfsdk:"enable_ipv4"`
	EnableIPv6             types.Bool   `tfsdk:"enable_ipv6"`

	// Shared server fields (updatable).
	Name                                 types.String `tfsdk:"name"`
	Description                          types.String `tfsdk:"description"`
	Port                                 types.Int64  `tfsdk:"port"`
	User                                 types.String `tfsdk:"user"`
	PrivateKeyUUID                       types.String `tfsdk:"private_key_uuid"`
	IsBuildServer                        types.Bool   `tfsdk:"is_build_server"`
	ConcurrentBuilds                     types.Int64  `tfsdk:"concurrent_builds"`
	DynamicTimeout                       types.Int64  `tfsdk:"dynamic_timeout"`
	DeploymentQueueLimit                 types.Int64  `tfsdk:"deployment_queue_limit"`
	ServerDiskUsageNotificationThreshold types.Int64  `tfsdk:"server_disk_usage_notification_threshold"`
	ServerDiskUsageCheckFrequency        types.String `tfsdk:"server_disk_usage_check_frequency"`
}

// NewResource returns a new Hetzner server resource.
func NewResource() resource.Resource {
	return &hetznerServerResource{}
}

func (r *hetznerServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_server"
}

func (r *hetznerServerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := hetznerSchemaAttributes()
	attrs["timeouts"] = timeouts.Attributes(ctx, timeouts.Opts{Create: true})
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provisions a Hetzner Cloud server and registers it with Coolify.\n\n~> **Warning:** Deleting this resource will delete the server from Coolify and cascade-delete all applications, databases, and services deployed on it. The underlying Hetzner Cloud server is not destroyed; manage its lifecycle separately.",
		Attributes:          attrs,
	}
}

func hetznerSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		// Computed after create.
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the server.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"ip": schema.StringAttribute{
			MarkdownDescription: "The IP address assigned to the server by Hetzner.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"is_reachable": schema.BoolAttribute{
			MarkdownDescription: "Whether the server is currently reachable.",
			Computed:            true,
		},
		"is_usable": schema.BoolAttribute{
			MarkdownDescription: "Whether the server is currently usable for deployments.",
			Computed:            true,
		},

		// Hetzner create-only fields.
		"cloud_provider_token_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the Hetzner cloud provider token (from `coolify_cloud_token`). Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"server_type": schema.StringAttribute{
			MarkdownDescription: "The Hetzner server type (e.g., `cx22`, `cpx31`). Use `coolify_hetzner_server_types` data source to list available types. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"location": schema.StringAttribute{
			MarkdownDescription: "The Hetzner datacenter location (e.g., `fsn1`, `nbg1`). Use `coolify_hetzner_locations` data source to list available locations. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"image": schema.StringAttribute{
			MarkdownDescription: "The OS image to use (e.g., `ubuntu-24.04`). Use `coolify_hetzner_images` data source to list available images. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"hetzner_ssh_key_ids": schema.StringAttribute{
			MarkdownDescription: "Comma-separated list of Hetzner SSH key IDs to install on the server. Use `coolify_hetzner_ssh_keys` data source to list available keys. Changing this forces a new resource.",
			Optional:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"cloud_init_script": schema.StringAttribute{
			MarkdownDescription: "Cloud-init script to run on server creation. Changing this forces a new resource.",
			Optional:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"instant_validate": schema.BoolAttribute{
			MarkdownDescription: "Whether to validate server connectivity immediately after creation.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"enable_ipv4": schema.BoolAttribute{
			MarkdownDescription: "Whether to enable IPv4 on the server.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"enable_ipv6": schema.BoolAttribute{
			MarkdownDescription: "Whether to enable IPv6 on the server.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},

		// Shared server fields (updatable).
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the server.",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "A description of the server.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"port": schema.Int64Attribute{
			MarkdownDescription: "The SSH port of the server.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(22),
			Validators:          []validator.Int64{int64validator.Between(1, 65535)},
		},
		"user": schema.StringAttribute{
			MarkdownDescription: "The SSH user for connecting to the server.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("root"),
		},
		"private_key_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the private key used for SSH authentication.",
			Required:            true,
			Validators:          []validator.String{validate.UUID()},
		},
		"is_build_server": schema.BoolAttribute{
			MarkdownDescription: "Whether this server is used for building applications.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"concurrent_builds": schema.Int64Attribute{
			MarkdownDescription: "How many deployments can run in parallel on this server.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(2),
			Validators:          []validator.Int64{int64validator.AtLeast(1)},
		},
		"dynamic_timeout": schema.Int64Attribute{
			MarkdownDescription: "Timeout in seconds for Docker operations (pull, build, health check) during deployment.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(3600),
			Validators:          []validator.Int64{int64validator.AtLeast(1)},
		},
		"deployment_queue_limit": schema.Int64Attribute{
			MarkdownDescription: "Maximum number of queued deployments (default 25).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(25),
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"server_disk_usage_notification_threshold": schema.Int64Attribute{
			MarkdownDescription: "Disk usage percentage at which a notification is sent.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(80),
			Validators:          []validator.Int64{int64validator.Between(1, 100)},
		},
		"server_disk_usage_check_frequency": schema.StringAttribute{
			MarkdownDescription: "Cron expression for how often disk usage is checked (e.g., `*/5 * * * *` or `@daily`).",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators: []validator.String{
				stringvalidator.RegexMatches(
					regexp.MustCompile(`^(\S+\s+){4}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
					"must be a valid cron expression (e.g., \"*/5 * * * *\" or \"@daily\")",
				),
			},
		},
	}
}

func (r *hetznerServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *hetznerServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hetznerServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_hetzner_server"})

	input := client.CreateHetznerServerInput{
		Name:                   plan.Name.ValueString(),
		CloudProviderTokenUUID: plan.CloudProviderTokenUUID.ValueString(),
		ServerType:             plan.ServerType.ValueString(),
		Location:               plan.Location.ValueString(),
		Image:                  plan.Image.ValueString(),
		PrivateKeyUUID:         plan.PrivateKeyUUID.ValueString(),
		EnableIPv4:             flex.BoolValueOrNull(plan.EnableIPv4),
		EnableIPv6:             flex.BoolValueOrNull(plan.EnableIPv6),
		InstantValidate:        flex.BoolValueOrNull(plan.InstantValidate),
	}
	flex.SetIfKnown(&input.HetznerSSHKeyIDs, plan.HetznerSSHKeyIDs)
	flex.SetIfKnown(&input.CloudInitScript, plan.CloudInitScript)

	created, err := r.client.CreateHetznerServer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Hetzner server",
			fmt.Sprintf("server %q: %s", plan.Name.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}
	if plan.IP.IsUnknown() {
		plan.IP = types.StringNull()
	}
	if plan.IsReachable.IsUnknown() {
		plan.IsReachable = types.BoolNull()
	}
	if plan.IsUsable.IsUnknown() {
		plan.IsUsable = types.BoolNull()
	}
	if plan.ServerDiskUsageCheckFrequency.IsUnknown() {
		plan.ServerDiskUsageCheckFrequency = types.StringNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply settings via PATCH if the user configured non-default values.
	if hasNonDefaultSettings(plan) {
		settingsUpdate := client.UpdateServerInput{
			ConcurrentBuilds:                     flex.IntIfNonDefault(plan.ConcurrentBuilds, 2),
			DynamicTimeout:                       flex.IntIfNonDefault(plan.DynamicTimeout, 3600),
			DeploymentQueueLimit:                 flex.IntIfNonDefault(plan.DeploymentQueueLimit, 25),
			ServerDiskUsageNotificationThreshold: flex.IntIfNonDefault(plan.ServerDiskUsageNotificationThreshold, 80),
			ServerDiskUsageCheckFrequency:        flex.StringValueOrNull(plan.ServerDiskUsageCheckFrequency),
		}
		if _, err := r.client.UpdateServer(ctx, created.UUID, settingsUpdate); err != nil {
			resp.Diagnostics.AddError("Error setting Hetzner server settings",
				fmt.Sprintf("server %s: %s", created.UUID, err))
			return
		}
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Hetzner server created but refresh failed",
			fmt.Sprintf("Coolify created Hetzner server %s, but the provider could not read it back: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, err),
		)
		return
	}

	flattenHetznerServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_hetzner_server", "uuid": created.UUID})
}

func (r *hetznerServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hetznerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_hetzner_server", "uuid": state.UUID.ValueString()})

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_hetzner_server", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Hetzner server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenHetznerServer(srv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hetznerServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hetznerServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state hetznerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_hetzner_server", "uuid": state.UUID.ValueString()})

	input := client.UpdateServerInput{
		Name:                                 flex.StringIfChanged(plan.Name, state.Name),
		Description:                          flex.StringIfChanged(plan.Description, state.Description),
		IP:                                   flex.StringIfChanged(plan.IP, state.IP),
		Port:                                 flex.IntIfChanged(plan.Port, state.Port),
		User:                                 flex.StringIfChanged(plan.User, state.User),
		PrivateKeyUUID:                       flex.StringIfChanged(plan.PrivateKeyUUID, state.PrivateKeyUUID),
		IsBuildServer:                        flex.BoolIfChanged(plan.IsBuildServer, state.IsBuildServer),
		ConcurrentBuilds:                     flex.IntIfChanged(plan.ConcurrentBuilds, state.ConcurrentBuilds),
		DynamicTimeout:                       flex.IntIfChanged(plan.DynamicTimeout, state.DynamicTimeout),
		DeploymentQueueLimit:                 flex.IntIfChanged(plan.DeploymentQueueLimit, state.DeploymentQueueLimit),
		ServerDiskUsageNotificationThreshold: flex.IntIfChanged(plan.ServerDiskUsageNotificationThreshold, state.ServerDiskUsageNotificationThreshold),
		ServerDiskUsageCheckFrequency:        flex.StringIfChanged(plan.ServerDiskUsageCheckFrequency, state.ServerDiskUsageCheckFrequency),
	}

	if _, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating Hetzner server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Hetzner server after update", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenHetznerServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hetznerServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hetznerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_hetzner_server", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting Hetzner server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_hetzner_server", "uuid": state.UUID.ValueString()})
}

func (r *hetznerServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func hasNonDefaultSettings(plan hetznerServerResourceModel) bool {
	intNonDefault := func(v types.Int64, dflt int64) bool {
		return !v.IsNull() && !v.IsUnknown() && v.ValueInt64() != dflt
	}
	strSet := func(v types.String) bool {
		return !v.IsNull() && !v.IsUnknown()
	}
	return intNonDefault(plan.ConcurrentBuilds, 2) ||
		intNonDefault(plan.DynamicTimeout, 3600) ||
		intNonDefault(plan.DeploymentQueueLimit, 25) ||
		intNonDefault(plan.ServerDiskUsageNotificationThreshold, 80) ||
		strSet(plan.ServerDiskUsageCheckFrequency)
}

func flattenHetznerServer(srv *client.Server, model *hetznerServerResourceModel) {
	model.UUID = types.StringValue(srv.UUID)
	model.Name = types.StringValue(srv.Name)
	model.Description = flex.StringToFramework(srv.Description)
	model.IP = types.StringValue(srv.IP)
	model.Port = types.Int64Value(int64(srv.Port))
	model.User = types.StringValue(srv.User)
	if srv.PrivateKeyUUID != "" {
		model.PrivateKeyUUID = types.StringValue(srv.PrivateKeyUUID)
	}
	model.IsBuildServer = types.BoolValue(srv.IsBuildServer)
	model.IsReachable = types.BoolValue(srv.IsReachable)
	model.IsUsable = types.BoolValue(srv.IsUsable)

	if srv.Settings != nil {
		model.ConcurrentBuilds = types.Int64Value(int64(srv.Settings.ConcurrentBuilds))
		model.DynamicTimeout = types.Int64Value(int64(srv.Settings.DynamicTimeout))
		model.DeploymentQueueLimit = types.Int64Value(int64(srv.Settings.DeploymentQueueLimit))
		model.ServerDiskUsageNotificationThreshold = types.Int64Value(int64(srv.Settings.ServerDiskUsageNotificationThreshold))
		model.ServerDiskUsageCheckFrequency = flex.StringToFramework(srv.Settings.ServerDiskUsageCheckFrequency)
	}
}
