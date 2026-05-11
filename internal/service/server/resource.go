package server

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	UUID           types.String `tfsdk:"uuid"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	IP             types.String `tfsdk:"ip"`
	Port           types.Int64  `tfsdk:"port"`
	User           types.String `tfsdk:"user"`
	PrivateKeyUUID types.String `tfsdk:"private_key_uuid"`
	IsBuildServer  types.Bool   `tfsdk:"is_build_server"`
	IsReachable    types.Bool   `tfsdk:"is_reachable"`
	IsUsable       types.Bool   `tfsdk:"is_usable"`
}

// NewResource returns a new server resource.
func NewResource() resource.Resource {
	return &serverResource{}
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify server.",
		Attributes: map[string]schema.Attribute{
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
			"Unexpected Resource Configure Type",
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
		resp.Diagnostics.AddError("Error creating server", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading server after create", err.Error())
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

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server", err.Error())
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

	port := int(plan.Port.ValueInt64())
	buildSrv := plan.IsBuildServer.ValueBool()

	input := client.UpdateServerInput{
		Port:          &port,
		IsBuildServer: &buildSrv,
	}
	flex.SetStrPtr(&input.Name, plan.Name)
	flex.SetStrPtr(&input.Description, plan.Description)
	flex.SetStrPtr(&input.IP, plan.IP)
	flex.SetStrPtr(&input.User, plan.User)
	flex.SetStrPtr(&input.PrivateKeyUUID, plan.PrivateKeyUUID)

	_, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating server", err.Error())
		return
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading server after update", err.Error())
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

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting server", err.Error())
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

func flattenServer(srv *client.Server, model *serverResourceModel) {
	model.UUID = types.StringValue(srv.UUID)
	model.Name = types.StringValue(srv.Name)
	model.Description = flex.StringToFramework(srv.Description)
	model.IP = types.StringValue(srv.IP)
	model.Port = types.Int64Value(int64(srv.Port))
	model.User = types.StringValue(srv.User)
	model.PrivateKeyUUID = types.StringValue(srv.PrivateKeyUUID)
	model.IsBuildServer = types.BoolValue(srv.IsBuildServer)
	model.IsReachable = types.BoolValue(srv.IsReachable)
	model.IsUsable = types.BoolValue(srv.IsUsable)
}
