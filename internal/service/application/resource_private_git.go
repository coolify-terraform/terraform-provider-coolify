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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &privateGitApplicationResource{}
	_ resource.ResourceWithConfigure   = &privateGitApplicationResource{}
	_ resource.ResourceWithImportState = &privateGitApplicationResource{}
)

// privateGitApplicationResource manages a Coolify application deployed from a private Git repository.
type privateGitApplicationResource struct {
	client *client.Client
}

type privateGitApplicationResourceModel struct {
	applicationCommonModel
	GitRepository      types.String `tfsdk:"git_repository"`
	GitBranch          types.String `tfsdk:"git_branch"`
	PrivateKeyUUID     types.String `tfsdk:"private_key_uuid"`
	BuildPack          types.String `tfsdk:"build_pack"`
	DockerfileLocation types.String `tfsdk:"dockerfile_location"`
	BuildCommand       types.String `tfsdk:"build_command"`
}

// NewPrivateGitResource returns a new privateGitApplicationResource instance.
func NewPrivateGitResource() resource.Resource {
	return &privateGitApplicationResource{}
}

func (r *privateGitApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_git_application"
}

func (r *privateGitApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a private Git repository using a deploy key.",
		Attributes: gitAppAttrs(ctx, "The Git SSH URL for the private repository (e.g. `git@github.com:org/repo.git`).", map[string]schema.Attribute{
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the SSH private key used for Git clone authentication. Changing this forces a new resource.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		}),
	}
}

func (r *privateGitApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

//nolint:dupl // Create methods differ by input struct type and API call
func (r *privateGitApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateGitApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_private_git_application"})

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
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.GitBranch, plan.GitBranch)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)
	flex.SetIfKnown(&input.DockerfileLocation, plan.DockerfileLocation)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.BuildCommand, plan.BuildCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreatePrivateGitApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating private git application",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	normalizeCommonAppCreateState(&plan.applicationCommonModel)
	normalizeUnknownString(&plan.GitBranch)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app := readBackAfterCreate(ctx, r.client, created.UUID, resp)
	if app == nil {
		return
	}

	flattenPrivateGitApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_private_git_application", "uuid": created.UUID})
}

func (r *privateGitApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateGitApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readApplication(ctx, r.client, "coolify_private_git_application", state.UUID.ValueString(), resp, func(app *client.Application) {
		flattenPrivateGitApplication(app, &state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	})
}

func (r *privateGitApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privateGitApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state privateGitApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_private_git_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common(), state.common())
	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenPrivateGitApplication(app, &plan)
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateGitApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateGitApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteApplication(ctx, r.client, "coolify_private_git_application", state.UUID.ValueString(), resp)
}

func (r *privateGitApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importApplicationState(ctx, req, resp)
}

func (m *privateGitApplicationResourceModel) common() commonAppFields {
	c := m.applicationCommonModel.common()
	c.GitRepository = &m.GitRepository
	c.GitBranch = &m.GitBranch
	c.BuildPack = &m.BuildPack
	c.DockerfileLocation = &m.DockerfileLocation
	c.BuildCommand = &m.BuildCommand
	return c
}

func flattenPrivateGitApplication(app *client.Application, state *privateGitApplicationResourceModel) {
	flattenApplicationCommon(app, state.common())
	if app.PrivateKeyUUID != "" {
		state.PrivateKeyUUID = types.StringValue(app.PrivateKeyUUID)
	}
}
