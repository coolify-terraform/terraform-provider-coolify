package application

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
	_ resource.Resource                = &dockerImageApplicationResource{}
	_ resource.ResourceWithConfigure   = &dockerImageApplicationResource{}
	_ resource.ResourceWithImportState = &dockerImageApplicationResource{}
)

// dockerImageApplicationResource manages a Coolify application deployed from a Docker image.
type dockerImageApplicationResource struct {
	client *client.Client
}

// dockerImageApplicationResourceModel maps the resource schema to Go types.
type dockerImageApplicationResourceModel struct {
	UUID            types.String   `tfsdk:"uuid"`
	Name            types.String   `tfsdk:"name"`
	Description     types.String   `tfsdk:"description"`
	ProjectUUID     types.String   `tfsdk:"project_uuid"`
	ServerUUID      types.String   `tfsdk:"server_uuid"`
	EnvironmentName types.String   `tfsdk:"environment_name"`
	DockerImage     types.String   `tfsdk:"docker_image"`
	PortsExposes    types.String   `tfsdk:"ports_exposes"`
	FQDN            types.String   `tfsdk:"fqdn"`
	InstallCommand  types.String   `tfsdk:"install_command"`
	StartCommand    types.String   `tfsdk:"start_command"`
	Status          types.String   `tfsdk:"status"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

// NewDockerResource returns a new dockerImageApplicationResource instance.
func NewDockerResource() resource.Resource {
	return &dockerImageApplicationResource{}
}

func (r *dockerImageApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_image_application"
}

func (r *dockerImageApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Docker image.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
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
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this application belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to deploy the application on. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
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
				MarkdownDescription: "The ports to expose, as a comma-separated list (e.g. `80` or `80,443`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g. \"80\" or \"80,443\")"),
				},
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{validate.FQDN()},
			},
			"install_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the install phase.",
				Optional:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "The command to run to start the application.",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the application (e.g. running, stopped, exited). Read-only.",
				Computed:            true,
			},
		},
	}
}

func (r *dockerImageApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dockerImageApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dockerImageApplicationResourceModel
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

	input := client.CreateDockerImageAppInput{
		ProjectUUID:  plan.ProjectUUID.ValueString(),
		ServerUUID:   plan.ServerUUID.ValueString(),
		DockerImage:  plan.DockerImage.ValueString(),
		PortsExposes: plan.PortsExposes.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateDockerImageApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating docker image application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after creation", err.Error())
		return
	}

	flattenDockerImageApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerImageApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerImageApplicationResourceModel
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

	flattenDockerImageApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dockerImageApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateApplicationInput{}
	strPtr := flex.StringValueOrNull
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

	flattenDockerImageApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerImageApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dockerImageApplicationResourceModel
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

func (r *dockerImageApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
}

// flattenDockerImageApplication copies API fields into the Terraform state model.
func flattenDockerImageApplication(app *client.Application, state *dockerImageApplicationResourceModel) {
	state.UUID = types.StringValue(app.UUID)
	state.Name = types.StringValue(app.Name)
	state.Description = flex.StringToFramework(app.Description)
	state.DockerImage = types.StringValue(app.DockerRegistryImageName)
	state.PortsExposes = types.StringValue(app.PortsExposes)
	state.FQDN = flex.StringToFramework(app.FQDN)
	state.InstallCommand = flex.StringToFramework(app.InstallCommand)
	state.StartCommand = flex.StringToFramework(app.StartCommand)
	state.Status = flex.StringToFramework(app.Status)

	if app.ProjectUUID != "" {
		state.ProjectUUID = types.StringValue(app.ProjectUUID)
	}
	if app.ServerUUID != "" {
		state.ServerUUID = types.StringValue(app.ServerUUID)
	}
	if app.EnvironmentName != "" {
		state.EnvironmentName = flex.StringToFramework(app.EnvironmentName)
	}
}
