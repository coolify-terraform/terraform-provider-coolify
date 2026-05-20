package application

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	resp.TypeName = req.ProviderTypeName + "_application_github_app"
}

func (r *gitHubAppApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed via a GitHub App integration. Coolify verifies repository access during create, so the referenced GitHub App must have installation access to the target repository.",
		Attributes: gitAppAttrs(ctx, "The Git repository URL (for example `https://github.com/org/repo` or `org/repo`). Coolify checks repository access during create.", map[string]schema.Attribute{
			"github_app_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the GitHub App used for repository access. The app installation must be able to read the repository configured in `git_repository`.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
		}),
	}
}

func (r *gitHubAppApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

//nolint:dupl // Create methods differ by input struct type and API call
func (r *gitHubAppApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_application_github_app"})

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
	flex.SetIfKnown(&input.Domains, plan.Domains)
	input.InstantDeploy = flex.BoolValueOrNull(plan.InstantDeploy)
	flex.SetIfKnown(&input.DockerfileLocation, plan.DockerfileLocation)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.BuildCommand, plan.BuildCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateGitHubAppApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitHub App application",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	normalizeCommonAppCreateState(&plan.applicationCommonModel)
	flex.NormalizeUnknownString(&plan.GitBranch)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	postCreatePatchExtendedFields(ctx, r.client, created.UUID, plan.common(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	app := readBackAfterCreate(ctx, r.client, created.UUID, resp)
	if app == nil {
		return
	}

	flattenGitHubAppApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_application_github_app", "uuid": created.UUID})
}

func (r *gitHubAppApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readApplication(ctx, r.client, "coolify_application_github_app", state.UUID.ValueString(), resp, func(app *client.Application) {
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_application_github_app", "uuid": plan.UUID.ValueString()})

	planFields := plan.common()
	stateFields := state.common()
	input := buildUpdateInput(planFields, stateFields)
	input.GitHubAppUUID = flex.StringIfChanged(plan.GitHubAppUUID, state.GitHubAppUUID)
	githubAppChanged := plan.GitHubAppUUID.ValueString() != state.GitHubAppUUID.ValueString()
	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenGitHubAppApplication(app, &plan)
	}, plan.RedeployOnUpdate.ValueBool(), planFields, stateFields, githubAppChanged)
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
	deleteApplication(ctx, r.client, "coolify_application_github_app", state.UUID.ValueString(), resp)
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
