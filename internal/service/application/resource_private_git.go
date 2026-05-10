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
	_ resource.Resource                = &PrivateGitApplicationResource{}
	_ resource.ResourceWithConfigure   = &PrivateGitApplicationResource{}
	_ resource.ResourceWithImportState = &PrivateGitApplicationResource{}
)

// PrivateGitApplicationResource manages a Coolify application deployed from a private Git repository.
type PrivateGitApplicationResource struct {
	client *client.Client
}

// PrivateGitApplicationResourceModel maps the resource schema to Go types.
type PrivateGitApplicationResourceModel struct {
	UUID               types.String   `tfsdk:"uuid"`
	Name               types.String   `tfsdk:"name"`
	Description        types.String   `tfsdk:"description"`
	ProjectUUID        types.String   `tfsdk:"project_uuid"`
	ServerUUID         types.String   `tfsdk:"server_uuid"`
	EnvironmentName    types.String   `tfsdk:"environment_name"`
	GitRepository      types.String   `tfsdk:"git_repository"`
	GitBranch          types.String   `tfsdk:"git_branch"`
	PrivateKeyUUID     types.String   `tfsdk:"private_key_uuid"`
	BuildPack          types.String   `tfsdk:"build_pack"`
	PortsExposes       types.String   `tfsdk:"ports_exposes"`
	FQDN               types.String   `tfsdk:"fqdn"`
	DockerfileLocation types.String   `tfsdk:"dockerfile_location"`
	InstallCommand     types.String   `tfsdk:"install_command"`
	BuildCommand       types.String   `tfsdk:"build_command"`
	StartCommand       types.String   `tfsdk:"start_command"`
	Status             types.String   `tfsdk:"status"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

// NewPrivateGitResource returns a new PrivateGitApplicationResource instance.
func NewPrivateGitResource() resource.Resource {
	return &PrivateGitApplicationResource{}
}

func (r *PrivateGitApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_git_application"
}

func (r *PrivateGitApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a private Git repository using a deploy key.",
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
			"git_repository": schema.StringAttribute{
				MarkdownDescription: "The Git SSH URL for the private repository (e.g. `git@github.com:org/repo.git`).",
				Required:            true,
			},
			"git_branch": schema.StringAttribute{
				MarkdownDescription: "The Git branch to deploy (defaults to `main`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("main"),
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the SSH private key used for Git clone authentication.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"build_pack": schema.StringAttribute{
				MarkdownDescription: "The build pack type. Valid values: `nixpacks`, `dockerfile`, `dockercompose`, `static`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("nixpacks", "dockerfile", "dockercompose", "static"),
				},
			},
			"ports_exposes": schema.StringAttribute{
				MarkdownDescription: "The ports to expose, as a comma-separated list (e.g. `3000` or `3000,8080`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g. \"3000\" or \"3000,8080\")"),
				},
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^https?://`), "must start with http:// or https://"),
				},
			},
			"dockerfile_location": schema.StringAttribute{
				MarkdownDescription: "The path to the Dockerfile, relative to the repository root.",
				Optional:            true,
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
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the application (e.g. running, stopped, exited). Read-only.",
				Computed:            true,
			},
		},
	}
}

func (r *PrivateGitApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrivateGitApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PrivateGitApplicationResourceModel
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

	input := client.CreatePrivateGitAppInput{
		ProjectUUID:    plan.ProjectUUID.ValueString(),
		ServerUUID:     plan.ServerUUID.ValueString(),
		GitRepository:  plan.GitRepository.ValueString(),
		BuildPack:      plan.BuildPack.ValueString(),
		PortsExposes:   plan.PortsExposes.ValueString(),
		PrivateKeyUUID: plan.PrivateKeyUUID.ValueString(),
	}
	if v := flex.StringFromFramework(plan.EnvironmentName); v != "" {
		input.EnvironmentName = v
	}
	if v := flex.StringFromFramework(plan.GitBranch); v != "" {
		input.GitBranch = v
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
	if v := flex.StringFromFramework(plan.DockerfileLocation); v != "" {
		input.DockerfileLocation = v
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

	created, err := r.client.CreatePrivateGitApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating private git application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	app, err := r.client.GetApplication(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after creation", err.Error())
		return
	}

	flattenPrivateGitApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PrivateGitApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PrivateGitApplicationResourceModel
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

	flattenPrivateGitApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PrivateGitApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PrivateGitApplicationResourceModel
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
	input.GitRepository = strPtr(plan.GitRepository)
	input.GitBranch = strPtr(plan.GitBranch)
	input.BuildPack = strPtr(plan.BuildPack)
	input.PortsExposes = strPtr(plan.PortsExposes)
	input.FQDN = strPtr(plan.FQDN)
	input.DockerfileLocation = strPtr(plan.DockerfileLocation)
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

	flattenPrivateGitApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PrivateGitApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PrivateGitApplicationResourceModel
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

func (r *PrivateGitApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
}

// flattenPrivateGitApplication copies API fields into the Terraform state model.
func flattenPrivateGitApplication(app *client.Application, state *PrivateGitApplicationResourceModel) {
	state.UUID = types.StringValue(app.UUID)
	state.Name = types.StringValue(app.Name)
	state.Description = flex.StringToFramework(app.Description)
	state.GitRepository = types.StringValue(app.GitRepository)
	state.GitBranch = types.StringValue(app.GitBranch)
	state.BuildPack = types.StringValue(app.BuildPack)
	state.PortsExposes = types.StringValue(app.PortsExposes)
	state.FQDN = flex.StringToFramework(app.FQDN)
	state.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	state.InstallCommand = flex.StringToFramework(app.InstallCommand)
	state.BuildCommand = flex.StringToFramework(app.BuildCommand)
	state.StartCommand = flex.StringToFramework(app.StartCommand)
	state.Status = flex.StringToFramework(app.Status)

	state.ProjectUUID = flex.StringToFramework(app.ProjectUUID)
	state.ServerUUID = flex.StringToFramework(app.ServerUUID)
}
