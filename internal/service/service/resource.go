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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

type serviceResource struct {
	client *client.Client
}

type serviceResourceModel struct {
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
	UUID             types.String   `tfsdk:"uuid"`
	Name             types.String   `tfsdk:"name"`
	Description      types.String   `tfsdk:"description"`
	ProjectUUID      types.String   `tfsdk:"project_uuid"`
	ServerUUID       types.String   `tfsdk:"server_uuid"`
	EnvironmentName  types.String   `tfsdk:"environment_name"`
	Type             types.String   `tfsdk:"type"`
	Status           types.String   `tfsdk:"status"`
	DockerCompose    types.String   `tfsdk:"docker_compose"`
	DockerComposeRaw types.String   `tfsdk:"docker_compose_raw"`
	ConnectToNetwork types.Bool     `tfsdk:"connect_to_docker_network"`
	ConfigHash       types.String   `tfsdk:"config_hash"`
}

func NewResource() resource.Resource {
	return &serviceResource{}
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service resource on Coolify. Services are pre-built application stacks from the Coolify service catalog (e.g., plausible, uptime-kuma, minio).",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this service belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to deploy the service on.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name. Defaults to `production`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("production"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The service type from the Coolify service catalog (e.g., `plausible`, `uptime-kuma`, `minio`). See the full list in the Coolify UI under Services > New Service, or in the [Coolify source](https://github.com/coollabsio/coolify/tree/v4.x/templates/service).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the service (e.g., `running`, `stopped`, `exited`). Read-only.",
				Computed:            true,
			},
			"docker_compose": schema.StringAttribute{
				MarkdownDescription: "The parsed Docker Compose configuration. Requires API token with `read:sensitive` permission.",
				Computed:            true,
				Sensitive:           true,
			},
			"docker_compose_raw": schema.StringAttribute{
				MarkdownDescription: "The raw Docker Compose configuration. Requires API token with `read:sensitive` permission.",
				Computed:            true,
				Sensitive:           true,
			},
			"connect_to_docker_network": schema.BoolAttribute{
				MarkdownDescription: "Whether the service containers connect to the Coolify Docker network.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"config_hash": schema.StringAttribute{
				MarkdownDescription: "Hash of the current service configuration. Changes when the compose or settings are modified.",
				Computed:            true,
			},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
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

	input := client.CreateServiceInput{
		ServerUUID:      plan.ServerUUID.ValueString(),
		ProjectUUID:     plan.ProjectUUID.ValueString(),
		EnvironmentName: plan.EnvironmentName.ValueString(),
		Type:            plan.Type.ValueString(),
	}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	created, err := r.client.CreateService(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Name.IsUnknown() {
		plan.Name = types.StringNull()
	}
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.client.GetService(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Service created but refresh failed",
			fmt.Sprintf("Coolify created service %s, but the provider could not read it back: Could not read service %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, created.UUID, err),
		)
		return
	}

	flattenService(svc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": created.UUID})
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
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_service", "uuid": state.UUID.ValueString()})
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
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": uuid})

	input := client.UpdateServiceInput{
		Name:        flex.StringIfChanged(plan.Name, state.Name),
		Description: flex.StringIfChanged(plan.Description, state.Description),
	}
	if _, err := r.client.UpdateService(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("service %s: %s", uuid, err))
		return
	}

	svc, err := r.client.GetService(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading service after update", fmt.Sprintf("service %s: %s", uuid, err))
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

	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": uuid})

	if err := r.client.DeleteService(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting service", fmt.Sprintf("service %s: %s", uuid, err))
		return
	}
	if !client.PollUntilDeleted(ctx, func() error { _, err := r.client.GetService(ctx, uuid); return err }) {
		tflog.Warn(ctx, "resource may still exist after polling timeout", map[string]interface{}{"resource_type": "coolify_service", "uuid": uuid})
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": uuid})
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, compound, err := validate.ParseCompoundImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parsed.UUID)...)
	if compound {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parsed.ProjectUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), parsed.ServerUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parsed.EnvironmentName)...)
	}
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides docker_compose and docker_compose_raw unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}

func flattenService(svc *client.Service, model *serviceResourceModel) {
	model.UUID = types.StringValue(svc.UUID)
	model.Name = flex.StringToFramework(svc.Name)
	model.Description = flex.StringToFramework(svc.Description)
	model.Status = flex.StringToFramework(svc.Status)
	model.DockerCompose = flex.StringToFramework(svc.DockerCompose)
	model.DockerComposeRaw = flex.StringToFramework(svc.DockerComposeRaw)
	model.ConfigHash = flex.StringToFramework(svc.ConfigHash)
	if svc.ConnectToNetwork != nil {
		model.ConnectToNetwork = types.BoolValue(*svc.ConnectToNetwork)
	} else {
		model.ConnectToNetwork = types.BoolValue(false)
	}

	// Immutable fields: only update if the API returns them because
	// Coolify may omit these from the GET response.
	if svc.Type != "" {
		model.Type = types.StringValue(svc.Type)
	}
	if svc.ProjectUUID != "" {
		model.ProjectUUID = types.StringValue(svc.ProjectUUID)
	}
	if svc.ServerUUID != "" {
		model.ServerUUID = types.StringValue(svc.ServerUUID)
	}
	if svc.EnvironmentName != "" {
		model.EnvironmentName = flex.StringToFramework(svc.EnvironmentName)
	}
}
