package privatekey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &privateKeyResource{}
	_ resource.ResourceWithConfigure   = &privateKeyResource{}
	_ resource.ResourceWithImportState = &privateKeyResource{}
)

const (
	privateKeyDeleteRetryAttempts = 6
	privateKeyDeleteRetryDelay    = 5 * time.Second
)

type privateKeyResource struct {
	client *client.Client
}

type privateKeyResourceModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	PrivateKey   types.String `tfsdk:"private_key"`
	PublicKey    types.String `tfsdk:"public_key"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
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
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The PEM-encoded private key content.",
				Required:            true,
				Sensitive:           true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The public key derived from the private key. Read-only.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "The fingerprint of the private key. Read-only.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_git_related": schema.BoolAttribute{
				MarkdownDescription: "Whether this key is used for Git operations. Determined by the server.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
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
			"Unexpected Configure Type",
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_private_key"})

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

	plan.UUID = types.StringValue(created.UUID)
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}
	if plan.PublicKey.IsUnknown() {
		plan.PublicKey = types.StringNull()
	}
	if plan.Fingerprint.IsUnknown() {
		plan.Fingerprint = types.StringNull()
	}
	if plan.IsGitRelated.IsUnknown() {
		plan.IsGitRelated = types.BoolNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back for full state.
	key, err := r.client.GetPrivateKey(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Private key created but refresh failed",
			fmt.Sprintf("Coolify created private key %s, but the provider could not read it back: Could not read private key %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, created.UUID, err),
		)
		return
	}

	// Preserve the user's original private key value. Coolify encrypts
	// the key and may return it in a different format, causing a
	// sensitive attribute mismatch.
	plannedKey := plan.PrivateKey
	flattenPrivateKey(key, &plan)
	plan.PrivateKey = plannedKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_private_key", "uuid": state.UUID.ValueString()})

	key, err := r.client.GetPrivateKey(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading private key", fmt.Sprintf("private key %s: %s", state.UUID.ValueString(), err))
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_private_key", "uuid": state.UUID.ValueString()})

	input := client.UpdatePrivateKeyInput{
		Name:        flex.StringIfChanged(plan.Name, state.Name),
		Description: flex.StringIfChanged(plan.Description, state.Description),
		PrivateKey:  flex.StringValueOrNull(plan.PrivateKey), // Required by Coolify on every PATCH
	}

	_, err := r.client.UpdatePrivateKey(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating private key", fmt.Sprintf("private key %s: %s", state.UUID.ValueString(), err))
		return
	}

	// Read back for full state.
	key, err := r.client.GetPrivateKey(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading private key after update", fmt.Sprintf("private key %s: %s", state.UUID.ValueString(), err))
		return
	}

	plannedKey := plan.PrivateKey
	flattenPrivateKey(key, &plan)
	plan.PrivateKey = plannedKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_private_key", "uuid": state.UUID.ValueString()})

	// Coolify deletes applications asynchronously. When terraform destroy
	// runs, the app referencing this key may still be deleting. Retry for
	// up to 30 seconds on "in use" errors.
	uuid := state.UUID.ValueString()
	err := client.RetryDelete(ctx, privateKeyDeleteRetryAttempts, privateKeyDeleteRetryDelay,
		func() error { return r.client.DeletePrivateKey(ctx, uuid) },
		isPrivateKeyDeleteRetryable,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting private key", fmt.Sprintf("private key %s: %s", uuid, err))
	}
}

func (r *privateKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides the private_key field unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}

func isPrivateKeyDeleteRetryable(err error) bool {
	message := err.Error()

	return strings.Contains(message, "in use") || strings.Contains(message, "cannot be deleted")
}

func flattenPrivateKey(key *client.PrivateKey, model *privateKeyResourceModel) {
	model.UUID = types.StringValue(key.UUID)
	model.Name = types.StringValue(key.Name)
	model.Description = flex.StringToFramework(key.Description)
	// Preserve private key from state if the API does not return it (sensitive field).
	if key.PrivateKey != "" {
		model.PrivateKey = types.StringValue(key.PrivateKey)
	}
	model.PublicKey = flex.StringToFramework(key.PublicKey)
	model.Fingerprint = flex.StringToFramework(key.Fingerprint)
	model.IsGitRelated = types.BoolValue(key.IsGitRelated)
}
