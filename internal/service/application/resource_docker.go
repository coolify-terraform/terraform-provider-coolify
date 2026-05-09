package application

import (
	"context"
	"fmt"
	"regexp"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &DockerImageApplicationResource{}
	_ resource.ResourceWithConfigure   = &DockerImageApplicationResource{}
	_ resource.ResourceWithImportState = &DockerImageApplicationResource{}
)

// DockerImageApplicationResource manages a Coolify application deployed from a Docker image.
type DockerImageApplicationResource struct {
	client *client.Client
}

// DockerImageApplicationResourceModel maps the resource schema to Go types.
type DockerImageApplicationResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ProjectUUID     types.String `tfsdk:"project_uuid"`
	ServerUUID      types.String `tfsdk:"server_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	DockerImage     types.String `tfsdk:"docker_image"`
	PortsExposes    types.String `tfsdk:"ports_exposes"`
	FQDN            types.String `tfsdk:"fqdn"`
	InstallCommand  types.String `tfsdk:"install_command"`
	StartCommand    types.String `tfsdk:"start_command"`
}

// NewDockerResource returns a new DockerImageApplicationResource instance.
func NewDockerResource() resource.Resource {
	return &DockerImageApplicationResource{}
}

func (r *DockerImageApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_image_application"
}

func (r *DockerImageApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Docker image.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the application.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the application.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the application.",
				Optional:            true,
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this application belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to deploy the application on. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name for the application (defaults to `production`). Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("production"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"docker_image": schema.StringAttribute{
				MarkdownDescription: "The Docker image to deploy (e.g. `nginx:latest`, `ghcr.io/org/app:v1`).",
				Required:            true,
			},
			"ports_exposes": schema.StringAttribute{
				MarkdownDescription: "The ports to expose (for example `80`).",
				Required:            true,
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^https?://`), "must start with http:// or https://"),
				},
			},
			"install_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the install phase.",
				Optional:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "The command to run to start the application.",
				Optional:            true,
			},
		},
	}
}

func (r *DockerImageApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DockerImageApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateDockerImageAppInput{
		ProjectUUID:  plan.ProjectUUID.ValueString(),
		ServerUUID:   plan.ServerUUID.ValueString(),
		DockerImage:  plan.DockerImage.ValueString(),
		PortsExposes: plan.PortsExposes.ValueString(),
	}
	if v := flex.StringFromFramework(plan.EnvironmentName); v != "" {
		input.EnvironmentName = v
	}
	if v := flex.StringFromFramework(plan.Name); v != "" {
		input.Name = v
	}
	if v := flex.StringFromFramework(plan.Description); v != "" {
		input.Description = v
	}
	if v := flex.StringFromFramework(plan.FQDN); v != "" {
		input.FQDN = v
	}
	if v := flex.StringFromFramework(plan.InstallCommand); v != "" {
		input.InstallCommand = v
	}
	if v := flex.StringFromFramework(plan.StartCommand); v != "" {
		input.StartCommand = v
	}

	created, err := r.client.CreateDockerImageApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating docker image application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	app, err := r.client.GetApplication(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after creation", err.Error())
		return
	}

	mapDockerAppToState(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DockerImageApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", err.Error())
		return
	}

	mapDockerAppToState(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DockerImageApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateApplicationInput{}
	strPtr := func(v types.String) *string {
		return flex.StringValueOrNull(v)
	}
	input.Name = strPtr(plan.Name)
	input.Description = strPtr(plan.Description)
	input.PortsExposes = strPtr(plan.PortsExposes)
	input.FQDN = strPtr(plan.FQDN)
	input.InstallCommand = strPtr(plan.InstallCommand)
	input.StartCommand = strPtr(plan.StartCommand)
	input.DockerRegistryImageName = strPtr(plan.DockerImage)

	_, err := r.client.UpdateApplication(ctx, plan.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}

	app, err := r.client.GetApplication(ctx, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after update", err.Error())
		return
	}

	mapDockerAppToState(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DockerImageApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApplication(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", err.Error())
		return
	}
}

func (r *DockerImageApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
}

// mapDockerAppToState copies API fields into the Terraform state model.
func mapDockerAppToState(app *client.Application, state *DockerImageApplicationResourceModel) {
	state.UUID = types.StringValue(app.UUID)
	state.Name = types.StringValue(app.Name)
	state.Description = flex.StringToFramework(app.Description)
	state.DockerImage = types.StringValue(app.DockerRegistryImageName)
	state.PortsExposes = types.StringValue(app.PortsExposes)
	state.FQDN = flex.StringToFramework(app.FQDN)
	state.InstallCommand = flex.StringToFramework(app.InstallCommand)
	state.StartCommand = flex.StringToFramework(app.StartCommand)

	if app.ProjectUUID != "" {
		state.ProjectUUID = types.StringValue(app.ProjectUUID)
	}
	if app.ServerUUID != "" {
		state.ServerUUID = types.StringValue(app.ServerUUID)
	}
}
