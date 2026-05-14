package application

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &gitHubAppApplicationResource{}
	_ resource.ResourceWithConfigure   = &gitHubAppApplicationResource{}
	_ resource.ResourceWithImportState = &gitHubAppApplicationResource{}
)

// gitHubAppApplicationResource manages a Coolify application deployed via a GitHub App.
type gitHubAppApplicationResource struct {
	client *client.Client
}

type gitHubAppApplicationResourceModel struct {
	applicationCommonModel
	GitHubAppUUID      types.String `tfsdk:"github_app_uuid"`
	GitRepository      types.String `tfsdk:"git_repository"`
	GitBranch          types.String `tfsdk:"git_branch"`
	BuildPack          types.String `tfsdk:"build_pack"`
	DockerfileLocation types.String `tfsdk:"dockerfile_location"`
	BuildCommand       types.String `tfsdk:"build_command"`
}

// NewGitHubAppResource returns a new gitHubAppApplicationResource instance.
func NewGitHubAppResource() resource.Resource {
	return &gitHubAppApplicationResource{}
}

func (r *gitHubAppApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_application"
}

func (r *gitHubAppApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed via a GitHub App integration.",
		Attributes: CommonAppAttrs(ctx, map[string]schema.Attribute{
			"github_app_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the GitHub App used for repository access.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"git_repository": schema.StringAttribute{
				MarkdownDescription: "The Git repository URL (e.g. `github.com/org/repo`).",
				Required:            true,
			},
			"git_branch": schema.StringAttribute{
				MarkdownDescription: "The Git branch to deploy (defaults to `main`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("main"),
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
		}),
	}
}

func (r *gitHubAppApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

//nolint:dupl // Create methods differ by input struct type and API call
func (r *gitHubAppApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_github_app_application"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input := client.CreateGitHubAppInput{
		ProjectUUID:   plan.ProjectUUID.ValueString(),
		ServerUUID:    plan.ServerUUID.ValueString(),
		GitHubAppUUID: plan.GitHubAppUUID.ValueString(),
		GitRepository: plan.GitRepository.ValueString(),
		BuildPack:     plan.BuildPack.ValueString(),
		PortsExposes:  plan.PortsExposes.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.GitBranch, plan.GitBranch)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)
	flex.SetIfKnown(&input.DockerfileLocation, plan.DockerfileLocation)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.BuildCommand, plan.BuildCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateGitHubAppApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitHub App application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app := readBackAfterCreate(ctx, r.client, created.UUID, resp)
	if app == nil {
		return
	}

	flattenGitHubAppApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readApplication(ctx, r.client, "coolify_github_app_application", state.UUID.ValueString(), resp, func(app *client.Application) {
		flattenGitHubAppApplication(app, &state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	})
}

func (r *gitHubAppApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_github_app_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common(), state.common())
	input.GitHubAppUUID = flex.StringIfChanged(plan.GitHubAppUUID, state.GitHubAppUUID)
	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenGitHubAppApplication(app, &plan)
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteApplication(ctx, r.client, "coolify_github_app_application", state.UUID.ValueString(), resp)
}

func (r *gitHubAppApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importApplicationState(ctx, req, resp)
}

func (m *gitHubAppApplicationResourceModel) common() commonAppFields {
	c := m.applicationCommonModel.common()
	c.GitRepository = &m.GitRepository
	c.GitBranch = &m.GitBranch
	c.BuildPack = &m.BuildPack
	c.DockerfileLocation = &m.DockerfileLocation
	c.BuildCommand = &m.BuildCommand
	return c
}

func flattenGitHubAppApplication(app *client.Application, state *gitHubAppApplicationResourceModel) {
	flattenApplicationCommon(app, state.common())
	if app.GitHubAppUUID != "" {
		state.GitHubAppUUID = types.StringValue(app.GitHubAppUUID)
	}
}
