package privatekey

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &privateKeyResource{}
	_ resource.ResourceWithConfigure   = &privateKeyResource{}
	_ resource.ResourceWithImportState = &privateKeyResource{}
)

type privateKeyResource struct {
	client *client.Client
}

type privateKeyResourceModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	PrivateKey   types.String `tfsdk:"private_key"`
	IsGitRelated types.Bool   `tfsdk:"is_git_related"`
}

// NewResource returns a new private key resource.
func NewResource() resource.Resource {
	return &privateKeyResource{}
}

func (r *privateKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_key"
}

func (r *privateKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify private key used for SSH authentication.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the private key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the private key.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the private key.",
				Optional:            true,
				Computed:            true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The PEM-encoded private key content.",
				Required:            true,
				Sensitive:           true,
			},
			"is_git_related": schema.BoolAttribute{
				MarkdownDescription: "Whether this key is used for Git operations. Determined by the server.",
				Computed:            true,
			},
		},
	}
}

func (r *privateKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *privateKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreatePrivateKeyInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		PrivateKey:  plan.PrivateKey.ValueString(),
	}

	created, err := r.client.CreatePrivateKey(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating private key", err.Error())
		return
	}

	// Read back for full state.
	key, err := r.client.GetPrivateKey(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading private key after create", err.Error())
		return
	}

	flattenPrivateKey(key, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetPrivateKey(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading private key", err.Error())
		return
	}

	flattenPrivateKey(key, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privateKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	desc := plan.Description.ValueString()
	pk := plan.PrivateKey.ValueString()

	input := client.UpdatePrivateKeyInput{
		Name:        &name,
		Description: &desc,
		PrivateKey:  &pk,
	}

	_, err := r.client.UpdatePrivateKey(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating private key", err.Error())
		return
	}

	// Read back for full state.
	key, err := r.client.GetPrivateKey(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading private key after update", err.Error())
		return
	}

	flattenPrivateKey(key, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePrivateKey(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting private key", err.Error())
		return
	}
}

func (r *privateKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func flattenPrivateKey(key *client.PrivateKey, model *privateKeyResourceModel) {
	model.UUID = types.StringValue(key.UUID)
	model.Name = types.StringValue(key.Name)
	model.Description = flex.StringToFramework(key.Description)
	model.PrivateKey = types.StringValue(key.PrivateKey)
	model.IsGitRelated = types.BoolValue(key.IsGitRelated)
}
