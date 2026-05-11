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
	_ resource.Resource                = &dockerfileApplicationResource{}
	_ resource.ResourceWithConfigure   = &dockerfileApplicationResource{}
	_ resource.ResourceWithImportState = &dockerfileApplicationResource{}
)

// dockerfileApplicationResource manages a Coolify application deployed from a Dockerfile.
type dockerfileApplicationResource struct {
	client *client.Client
}

// dockerfileApplicationResourceModel maps the resource schema to Go types.
type dockerfileApplicationResourceModel struct {
	UUID               types.String   `tfsdk:"uuid"`
	Name               types.String   `tfsdk:"name"`
	Description        types.String   `tfsdk:"description"`
	ProjectUUID        types.String   `tfsdk:"project_uuid"`
	ServerUUID         types.String   `tfsdk:"server_uuid"`
	EnvironmentName    types.String   `tfsdk:"environment_name"`
	DockerfileLocation types.String   `tfsdk:"dockerfile_location"`
	PortsExposes       types.String   `tfsdk:"ports_exposes"`
	FQDN               types.String   `tfsdk:"fqdn"`
	InstallCommand     types.String   `tfsdk:"install_command"`
	BuildCommand       types.String   `tfsdk:"build_command"`
	StartCommand       types.String   `tfsdk:"start_command"`
	GitRepository      types.String   `tfsdk:"git_repository"`
	GitBranch          types.String   `tfsdk:"git_branch"`
	BuildPack          types.String   `tfsdk:"build_pack"`
	Status             types.String   `tfsdk:"status"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

// NewDockerfileResource returns a new dockerfileApplicationResource instance.
func NewDockerfileResource() resource.Resource {
	return &dockerfileApplicationResource{}
}

func (r *dockerfileApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dockerfile_application"
}

func (r *dockerfileApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Dockerfile.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
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
			"dockerfile_location": schema.StringAttribute{
				MarkdownDescription: "The path to the Dockerfile (e.g. `/Dockerfile`).",
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
				Validators:          []validator.String{validate.FQDN()},
			},
			"install_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the install phase.",
				Optional:            true,
			},
			"build_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the build phase.",
				Optional:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "The command to run to start the application.",
				Optional:            true,
			},
			"git_repository": schema.StringAttribute{
				MarkdownDescription: "The Git repository URL. Read-only, set by the API.",
				Computed:            true,
			},
			"git_branch": schema.StringAttribute{
				MarkdownDescription: "The Git branch. Read-only, set by the API.",
				Computed:            true,
			},
			"build_pack": schema.StringAttribute{
				MarkdownDescription: "The build pack type. Read-only, set by the API.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the application (e.g. running, stopped, exited). Read-only.",
				Computed:            true,
			},
		},
	}
}

func (r *dockerfileApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dockerfileApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dockerfileApplicationResourceModel
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

	input := client.CreateDockerfileAppInput{
		ProjectUUID:        plan.ProjectUUID.ValueString(),
		ServerUUID:         plan.ServerUUID.ValueString(),
		DockerfileLocation: plan.DockerfileLocation.ValueString(),
		PortsExposes:       plan.PortsExposes.ValueString(),
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
	if v := flex.StringFromFramework(plan.BuildCommand); v != "" {
		input.BuildCommand = v
	}
	if v := flex.StringFromFramework(plan.StartCommand); v != "" {
		input.StartCommand = v
	}

	created, err := r.client.CreateDockerfileApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dockerfile application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	app, err := r.client.GetApplication(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after creation", err.Error())
		return
	}

	flattenDockerfileApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerfileApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerfileApplicationResourceModel
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

	flattenDockerfileApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dockerfileApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerfileApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateApplicationInput{}
	strPtr := flex.StringValueOrNull
	input.Name = strPtr(plan.Name)
	input.Description = strPtr(plan.Description)
	input.DockerfileLocation = strPtr(plan.DockerfileLocation)
	input.PortsExposes = strPtr(plan.PortsExposes)
	input.FQDN = strPtr(plan.FQDN)
	input.InstallCommand = strPtr(plan.InstallCommand)
	input.BuildCommand = strPtr(plan.BuildCommand)
	input.StartCommand = strPtr(plan.StartCommand)

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

	flattenDockerfileApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerfileApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dockerfileApplicationResourceModel
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

func (r *dockerfileApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
}

// flattenDockerfileApplication copies API fields into the Terraform state model.
//
//nolint:dupl // shared flatten extraction tracked in #11
func flattenDockerfileApplication(app *client.Application, state *dockerfileApplicationResourceModel) {
	state.UUID = types.StringValue(app.UUID)
	state.Name = types.StringValue(app.Name)
	state.Description = flex.StringToFramework(app.Description)
	state.DockerfileLocation = types.StringValue(app.DockerfileLocation)
	state.PortsExposes = types.StringValue(app.PortsExposes)
	state.FQDN = flex.StringToFramework(app.FQDN)
	state.InstallCommand = flex.StringToFramework(app.InstallCommand)
	state.BuildCommand = flex.StringToFramework(app.BuildCommand)
	state.StartCommand = flex.StringToFramework(app.StartCommand)
	state.GitRepository = flex.StringToFramework(app.GitRepository)
	state.GitBranch = flex.StringToFramework(app.GitBranch)
	state.BuildPack = flex.StringToFramework(app.BuildPack)
	state.Status = flex.StringToFramework(app.Status)

	state.ProjectUUID = flex.StringToFramework(app.ProjectUUID)
	state.ServerUUID = flex.StringToFramework(app.ServerUUID)
}
