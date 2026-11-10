//nolint:dupl // gitea-app and github-app share CRUD shape; Create inputs differ
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &giteaAppApplicationResource{}
	_ resource.ResourceWithConfigure   = &giteaAppApplicationResource{}
	_ resource.ResourceWithImportState = &giteaAppApplicationResource{}
	_ resource.ResourceWithModifyPlan  = &giteaAppApplicationResource{}
)

// giteaAppApplicationResource manages a Coolify application deployed via a Gitea app.
type giteaAppApplicationResource struct {
	client *client.Client
}

type giteaAppApplicationResourceModel struct {
	applicationCommonModel
	GitRepository                   types.String `tfsdk:"git_repository"`
	GitBranch                       types.String `tfsdk:"git_branch"`
	BuildPack                       types.String `tfsdk:"build_pack"`
	DockerfileLocation              types.String `tfsdk:"dockerfile_location"`
	DockerComposeLocation           types.String `tfsdk:"docker_compose_location"`
	DockerComposeCustomStartCommand types.String `tfsdk:"docker_compose_custom_start_command"`
	DockerComposeCustomBuildCommand types.String `tfsdk:"docker_compose_custom_build_command"`
	DockerfileTargetBuild           types.String `tfsdk:"dockerfile_target_build"`
	BuildCommand                    types.String `tfsdk:"build_command"`
}

// NewGiteaAppResource returns a new giteaAppApplicationResource instance.
func NewGiteaAppResource() resource.Resource {
	return &giteaAppApplicationResource{}
}

func (r *giteaAppApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_gitea_app"
}

func (r *giteaAppApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed via a Gitea app. Coolify checks repository access during create, so the Gitea app must be connected and able to read `git_repository`. Use `owner/name` or an HTTPS URL on that Gitea host. Coolify stores `owner/name`, and the provider keeps the URL you configured. Changing `gitea_app_uuid` updates the source in place on Coolify >= v4.7.0. Older instances keep the value in state and do not send it.",
		Attributes: gitAppAttrs(ctx, "The Git repository URL (for example `https://gitea.example.com/org/repo` or `org/repo`).", map[string]schema.Attribute{
			"gitea_app_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the connected Gitea app that clones this repository. Requires Coolify >= v4.7.0 to change after create. Do not set it together with `github_app_uuid`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{validate.UUID()},
			},
		}),
	}
}

func (r *giteaAppApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *giteaAppApplicationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var cfg giteaAppApplicationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnUnsupportedApplicationSettingsWrites(r.client, cfg.common(), &resp.Diagnostics)
}

//nolint:dupl // Create methods differ by input struct type and API call
func (r *giteaAppApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan giteaAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !r.client.SupportsCoolify47() {
		ver := "unknown"
		if r.client != nil && r.client.CoolifyVersion != "" {
			ver = r.client.CoolifyVersion
		}
		resp.Diagnostics.AddError(
			"Coolify version cannot create a Gitea application",
			fmt.Sprintf("This Coolify instance (%s) has no POST /applications/private-gitea-app (added in v4.7.0).", ver),
		)
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_application_gitea_app"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input := client.CreateGiteaAppInput{
		ProjectUUID:   plan.ProjectUUID.ValueString(),
		ServerUUID:    plan.ServerUUID.ValueString(),
		GiteaAppUUID:  plan.GiteaAppUUID.ValueString(),
		GitRepository: plan.GitRepository.ValueString(),
		BuildPack:     plan.BuildPack.ValueString(),
		PortsExposes:  plan.PortsExposes.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.EnvironmentUUID, plan.EnvironmentUUID)
	flex.SetIfKnown(&input.DestinationUUID, plan.DestinationUUID)
	flex.SetIfKnown(&input.GitBranch, plan.GitBranch)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Domains, plan.Domains)
	input.InstantDeploy = flex.BoolValueOrNull(plan.InstantDeploy)
	input.AutogenerateDomain = flex.BoolValueOrNull(plan.AutogenerateDomain)
	input.ForceDomainOverride = flex.BoolValueOrNull(plan.ForceDomainOverride)
	flex.SetIfKnown(&input.DockerfileLocation, plan.DockerfileLocation)
	flex.SetIfKnown(&input.DockerComposeLocation, plan.DockerComposeLocation)
	flex.SetIfKnown(&input.DockerComposeCustomStartCommand, plan.DockerComposeCustomStartCommand)
	flex.SetIfKnown(&input.DockerComposeCustomBuildCommand, plan.DockerComposeCustomBuildCommand)
	if !plan.Dockerfile.IsNull() && !plan.Dockerfile.IsUnknown() {
		if df := plan.Dockerfile.ValueString(); df != "" {
			input.Dockerfile = flex.EnsureBase64(df)
		}
	}
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.BuildCommand, plan.BuildCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateGiteaAppApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Gitea application",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	normalizeCommonAppCreateState(&plan.applicationCommonModel)
	flex.NormalizeUnknownString(&plan.GitBranch)
	flex.NormalizeUnknownString(&plan.BuildCommand)
	flex.NormalizeUnknownString(&plan.DockerComposeCustomBuildCommand)
	flex.NormalizeUnknownString(&plan.DockerComposeCustomStartCommand)
	flex.NormalizeUnknownString(&plan.DockerComposeLocation)
	flex.NormalizeUnknownString(&plan.DockerfileTargetBuild)

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

	flattenGiteaAppApplication(app, &plan, r.client)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_application_gitea_app", "uuid": created.UUID})
}

func (r *giteaAppApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state giteaAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readApplication(ctx, r.client, "coolify_application_gitea_app", state.UUID.ValueString(), resp, func(app *client.Application) {
		flattenGiteaAppApplication(app, &state, r.client)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	})
}

func (r *giteaAppApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan giteaAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state giteaAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_application_gitea_app", "uuid": plan.UUID.ValueString()})

	planFields := plan.common()
	stateFields := state.common()
	input := buildUpdateInput(planFields, stateFields)
	updateAndReadBack(ctx, r.client, resp, updateAndReadBackArgs{
		ForceDomainOverride: forceDomainOverrideConfigured(ctx, req.Config, &resp.Diagnostics),
		UUID:                plan.UUID.ValueString(),
		Input:               input,
		Flatten: func(app *client.Application) {
			flattenGiteaAppApplication(app, &plan, r.client)
		},
		RedeployOnUpdate: plan.RedeployOnUpdate.ValueBool(),
		Plan:             planFields,
		State:            stateFields,
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *giteaAppApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state giteaAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteApplication(ctx, r.client, "coolify_application_gitea_app", state.UUID.ValueString(), state.Timeouts, resp)
}

func (r *giteaAppApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importApplicationState(ctx, r.client, req, resp)
}

func (m *giteaAppApplicationResourceModel) common() commonAppFields {
	c := m.applicationCommonModel.common()
	c.GitRepository = &m.GitRepository
	c.GitBranch = &m.GitBranch
	c.BuildPack = &m.BuildPack
	c.DockerfileLocation = &m.DockerfileLocation
	c.DockerComposeLocation = &m.DockerComposeLocation
	c.DockerComposeCustomStartCommand = &m.DockerComposeCustomStartCommand
	c.DockerComposeCustomBuildCommand = &m.DockerComposeCustomBuildCommand
	c.DockerfileTargetBuild = &m.DockerfileTargetBuild
	c.BuildCommand = &m.BuildCommand
	return c
}

func flattenGiteaAppApplication(app *client.Application, state *giteaAppApplicationResourceModel, c *client.Client) {
	f := state.common()
	noteCoolifyVersion(&f, c)
	flattenApplicationCommon(app, f)
}
