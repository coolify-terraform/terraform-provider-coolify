package service

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{}
)

type serviceResource struct{ client *client.Client }
type serviceResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ProjectUUID     types.String `tfsdk:"project_uuid"`
	ServerUUID      types.String `tfsdk:"server_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	Type            types.String `tfsdk:"type"`
}

func NewResource() resource.Resource { return &serviceResource{} }
func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}
func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service resource on Coolify. Services are pre-built application stacks from the Coolify service catalog (e.g. plausible, uptime-kuma, minio).",
		Attributes: map[string]schema.Attribute{
			"uuid":             schema.StringAttribute{MarkdownDescription: "The UUID of the service.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":             schema.StringAttribute{MarkdownDescription: "The name of the service.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"description":      schema.StringAttribute{MarkdownDescription: "A description of the service.", Optional: true},
			"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this service belongs to.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the service on.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"environment_name": schema.StringAttribute{MarkdownDescription: "The environment name. Defaults to `production`.", Optional: true, Computed: true, Default: stringdefault.StaticString("production")},
			"type":             schema.StringAttribute{MarkdownDescription: "The service type from the Coolify service catalog (e.g. `plausible`, `uptime-kuma`, `minio`).", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		},
	}
}
func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = c
}
func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := client.CreateServiceInput{ServerUUID: plan.ServerUUID.ValueString(), ProjectUUID: plan.ProjectUUID.ValueString(), EnvironmentName: plan.EnvironmentName.ValueString(), Type: plan.Type.ValueString()}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		input.Name = plan.Name.ValueString()
	}
	if !plan.Description.IsNull() {
		input.Description = plan.Description.ValueString()
	}
	created, err := r.client.CreateService(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}
	svc, err := r.client.GetService(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading service after creation", err.Error())
		return
	}
	plan.UUID = types.StringValue(svc.UUID)
	if svc.Name != "" {
		plan.Name = types.StringValue(svc.Name)
	}
	if svc.Description != "" {
		plan.Description = types.StringValue(svc.Description)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	svc, err := r.client.GetService(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}
	state.UUID = types.StringValue(svc.UUID)
	if svc.Name != "" {
		state.Name = types.StringValue(svc.Name)
	}
	if svc.Description != "" {
		state.Description = types.StringValue(svc.Description)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteService(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
	}
}
func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
