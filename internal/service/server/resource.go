package server

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
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithConfigure   = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}
)

type serverResource struct {
	client *client.Client
}

type serverResourceModel struct {
	Timeouts                             timeouts.Value `tfsdk:"timeouts"`
	UUID                                 types.String   `tfsdk:"uuid"`
	Name                                 types.String   `tfsdk:"name"`
	Description                          types.String   `tfsdk:"description"`
	IP                                   types.String   `tfsdk:"ip"`
	Port                                 types.Int64    `tfsdk:"port"`
	User                                 types.String   `tfsdk:"user"`
	PrivateKeyUUID                       types.String   `tfsdk:"private_key_uuid"`
	IsBuildServer                        types.Bool     `tfsdk:"is_build_server"`
	IsReachable                          types.Bool     `tfsdk:"is_reachable"`
	IsUsable                             types.Bool     `tfsdk:"is_usable"`
	ConcurrentBuilds                     types.Int64    `tfsdk:"concurrent_builds"`
	DynamicTimeout                       types.Int64    `tfsdk:"dynamic_timeout"`
	DeploymentQueueLimit                 types.Int64    `tfsdk:"deployment_queue_limit"`
	ServerDiskUsageNotificationThreshold types.Int64    `tfsdk:"server_disk_usage_notification_threshold"`
	ServerDiskUsageCheckFrequency        types.String   `tfsdk:"server_disk_usage_check_frequency"`
}

// NewResource returns a new server resource.
func NewResource() resource.Resource {
	return &serverResource{}
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify server.\n\n~> **Warning:** Deleting a server will cascade-delete all applications, databases, and services deployed on it.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the server.",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The SSH port of the server.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(22),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
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
			"is_reachable": schema.BoolAttribute{
				MarkdownDescription: "Whether the server is currently reachable.",
				Computed:            true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether the server is currently usable for deployments.",
				Computed:            true,
			},
			"concurrent_builds": schema.Int64Attribute{
				MarkdownDescription: "How many deployments can run in parallel on this server.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"dynamic_timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for Docker operations (pull, build, health check) during deployment.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
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
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"server_disk_usage_check_frequency": schema.StringAttribute{
				MarkdownDescription: "Cron expression for how often disk usage is checked (e.g. `*/5 * * * *` or `@daily`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\S+\s+){4}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
						"must be a valid cron expression (e.g. \"*/5 * * * *\" or \"@daily\")",
					),
				},
			},
		},
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_server"})

	input := client.CreateServerInput{
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		IP:             plan.IP.ValueString(),
		Port:           int(plan.Port.ValueInt64()),
		User:           plan.User.ValueString(),
		PrivateKeyUUID: plan.PrivateKeyUUID.ValueString(),
		IsBuildServer:  flex.BoolValueOrNull(plan.IsBuildServer),
	}

	created, err := r.client.CreateServer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating server",
			fmt.Sprintf("server %q (IP %s): %s", plan.Name.ValueString(), plan.IP.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
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

	// The Create endpoint only accepts core fields (name, ip, port, etc.).
	// Settings like concurrent_builds and dynamic_timeout must be sent via
	// a follow-up PATCH if the user configured non-default values.
	if hasNonDefaultSettings(plan) {
		settingsUpdate := client.UpdateServerInput{
			ConcurrentBuilds:                     flex.IntIfNonDefault(plan.ConcurrentBuilds, 2),
			DynamicTimeout:                       flex.IntIfNonDefault(plan.DynamicTimeout, 3600),
			DeploymentQueueLimit:                 flex.IntIfNonDefault(plan.DeploymentQueueLimit, 25),
			ServerDiskUsageNotificationThreshold: flex.IntIfNonDefault(plan.ServerDiskUsageNotificationThreshold, 80),
			ServerDiskUsageCheckFrequency:        flex.StringValueOrNull(plan.ServerDiskUsageCheckFrequency),
		}
		if _, err := r.client.UpdateServer(ctx, created.UUID, settingsUpdate); err != nil {
			resp.Diagnostics.AddError("Error setting server settings",
				fmt.Sprintf("server %s: %s", created.UUID, err))
			return
		}
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Server created but refresh failed",
			fmt.Sprintf("Coolify created server %s, but the provider could not read it back: Could not read server %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, created.UUID, err),
		)
		return
	}

	flattenServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenServer(srv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})

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

	_, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading server after update", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// hasNonDefaultSettings returns true if the user configured any settings
// field to a value different from Coolify's create-time default.
func hasNonDefaultSettings(plan serverResourceModel) bool {
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

func flattenServer(srv *client.Server, model *serverResourceModel) {
	model.UUID = types.StringValue(srv.UUID)
	model.Name = types.StringValue(srv.Name)
	model.Description = flex.StringToFramework(srv.Description)
	model.IP = types.StringValue(srv.IP)
	model.Port = types.Int64Value(int64(srv.Port))
	model.User = types.StringValue(srv.User)
	// Only overwrite private_key_uuid when the API returns a non-empty
	// value. The GET response omits this field, which would cause
	// "inconsistent result after apply" if we set it to "".
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
