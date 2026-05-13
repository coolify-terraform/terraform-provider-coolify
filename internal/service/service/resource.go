package service

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{}
)

type serviceResource struct{ client *client.Client }
type serviceResourceModel struct {
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	UUID            types.String   `tfsdk:"uuid"`
	Name            types.String   `tfsdk:"name"`
	Description     types.String   `tfsdk:"description"`
	ProjectUUID     types.String   `tfsdk:"project_uuid"`
	ServerUUID      types.String   `tfsdk:"server_uuid"`
	EnvironmentName types.String   `tfsdk:"environment_name"`
	Type            types.String   `tfsdk:"type"`
}

func NewResource() resource.Resource { return &serviceResource{} }
func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}
func (r *serviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service resource on Coolify. Services are pre-built application stacks from the Coolify service catalog (e.g. plausible, uptime-kuma, minio).",
		Attributes: map[string]schema.Attribute{
			"timeouts":         timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
			"uuid":             schema.StringAttribute{MarkdownDescription: "The UUID of the service.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":             schema.StringAttribute{MarkdownDescription: "The name of the service.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"description":      schema.StringAttribute{MarkdownDescription: "A description of the service.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this service belongs to.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the service on.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"environment_name": schema.StringAttribute{MarkdownDescription: "The environment name. Defaults to `production`. Changing this forces a new resource.", Optional: true, Computed: true, Default: stringdefault.StaticString("production"), PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"type":             schema.StringAttribute{MarkdownDescription: "The service type from the Coolify service catalog (e.g. `plausible`, `uptime-kuma`, `minio`). See the full list in the Coolify UI under Services > New Service, or in the [Coolify source](https://github.com/coollabsio/coolify/tree/v4.x/templates/service).", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		},
	}
}
func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
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
	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_service"})

	input := client.CreateServiceInput{ServerUUID: plan.ServerUUID.ValueString(), ProjectUUID: plan.ProjectUUID.ValueString(), EnvironmentName: plan.EnvironmentName.ValueString(), Type: plan.Type.ValueString()}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	created, err := r.client.CreateService(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.client.GetService(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading service after creation", fmt.Sprintf("service %s: %s", created.UUID, err))
		return
	}
	flattenService(svc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": state.UUID.ValueString()})

	svc, err := r.client.GetService(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service", fmt.Sprintf("service %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenService(svc, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": plan.UUID.ValueString()})

	input := client.UpdateServiceInput{}
	flex.SetStrPtr(&input.Name, plan.Name)
	flex.SetStrPtr(&input.Description, plan.Description)
	if _, err := r.client.UpdateService(ctx, plan.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("service %s: %s", plan.UUID.ValueString(), err))
		return
	}
	svc, err := r.client.GetService(ctx, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service after update", fmt.Sprintf("service %s: %s", plan.UUID.ValueString(), err))
		return
	}
	flattenService(svc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteService(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting service", fmt.Sprintf("service %s: %s", state.UUID.ValueString(), err))
		return
	}
}
func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides docker_compose and docker_compose_raw unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}

func flattenService(svc *client.Service, m *serviceResourceModel) {
	m.UUID = types.StringValue(svc.UUID)
	m.Name = flex.StringToFramework(svc.Name)
	m.Description = flex.StringToFramework(svc.Description)
	// Immutable fields: only update if the API returns them (Coolify may
	// omit these from the GET response).
	if svc.Type != "" {
		m.Type = types.StringValue(svc.Type)
	}
	if svc.ProjectUUID != "" {
		m.ProjectUUID = types.StringValue(svc.ProjectUUID)
	}
	if svc.ServerUUID != "" {
		m.ServerUUID = types.StringValue(svc.ServerUUID)
	}
	if svc.EnvironmentName != "" {
		m.EnvironmentName = flex.StringToFramework(svc.EnvironmentName)
	}
}
