package s3storage

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*s3StorageResource)(nil)
	_ resource.ResourceWithImportState = (*s3StorageResource)(nil)
	_ resource.ResourceWithConfigure   = (*s3StorageResource)(nil)
)

// s3StorageResource is the resource implementation for Coolify S3 storage.
type s3StorageResource struct {
	client *client.Client
}

// s3StorageResourceModel maps the resource schema data.
type s3StorageResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
	Key         types.String `tfsdk:"key"`
	Secret      types.String `tfsdk:"secret"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`
}

// NewResource returns a new S3 storage resource instance.
func NewResource() resource.Resource {
	return &s3StorageResource{}
}

func (r *s3StorageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

func (r *s3StorageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify S3-compatible storage configuration. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the S3 storage.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "A friendly name for the S3 storage.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the S3 storage.",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "S3-compatible endpoint URL (e.g. `https://s3.us-east-1.amazonaws.com`). " +
					"Coolify validates this with its SafeWebhookUrl rule: private, loopback, and most internal hostnames " +
					"(including Docker DNS names such as `coolify-minio`) are rejected unless the Coolify instance " +
					"explicitly allowlists them. Use a public HTTPS endpoint for API-managed storages.",
				Required: true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "S3 bucket name.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "S3 region (e.g. `us-east-1`).",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Access key. Sensitive; Coolify may omit it on read unless the API token can read sensitive fields. Preserve the value in configuration after import.",
				Required:            true,
				Sensitive:           true,
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Secret key. Sensitive; Coolify may omit it on read unless the API token can read sensitive fields. Preserve the value in configuration after import.",
				Required:            true,
				Sensitive:           true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify marks this storage as usable. Defaults to `false`. Connection validation can update this on the server.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *s3StorageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *s3StorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan s3StorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_s3_storage"})

	input := client.CreateS3StorageInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Endpoint:    plan.Endpoint.ValueString(),
		Bucket:      plan.Bucket.ValueString(),
		Region:      plan.Region.ValueString(),
		Key:         plan.Key.ValueString(),
		Secret:      plan.Secret.ValueString(),
		IsUsable:    flex.BoolValueOrNull(plan.IsUsable),
	}

	created, err := r.client.CreateS3Storage(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating S3 storage", fmt.Sprintf("s3 storage %q: %s", plan.Name.ValueString(), err))
		return
	}

	createdUUID := created.UUID
	plan.UUID = types.StringValue(createdUUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the full S3 storage to populate all fields.
	s, err := r.client.GetS3Storage(ctx, createdUUID)
	if err != nil {
		addCreateReadBackError(resp, createdUUID, err)
		return
	}

	flattenS3Storage(s, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_s3_storage", "uuid": plan.UUID.ValueString()})
}

func (r *s3StorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_s3_storage", "uuid": state.UUID.ValueString()})

	s, err := r.client.GetS3Storage(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_s3_storage", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading S3 storage", fmt.Sprintf("Could not read s3 storage %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenS3Storage(s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *s3StorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan s3StorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_s3_storage", "uuid": state.UUID.ValueString()})

	input := client.UpdateS3StorageInput{
		Name:        flex.StringIfChanged(plan.Name, state.Name),
		Description: flex.StringIfChanged(plan.Description, state.Description),
		Endpoint:    flex.StringIfChanged(plan.Endpoint, state.Endpoint),
		Bucket:      flex.StringIfChanged(plan.Bucket, state.Bucket),
		Region:      flex.StringIfChanged(plan.Region, state.Region),
		Key:         flex.StringIfChanged(plan.Key, state.Key),
		Secret:      flex.StringIfChanged(plan.Secret, state.Secret),
		IsUsable:    flex.BoolIfChanged(plan.IsUsable, state.IsUsable),
	}

	_, err := r.client.UpdateS3Storage(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating S3 storage", fmt.Sprintf("Could not update s3 storage %s: %s", state.UUID.ValueString(), err))
		return
	}

	plan.UUID = state.UUID

	// Read back the full S3 storage to populate all fields.
	s, err := r.client.GetS3Storage(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading S3 storage after update", fmt.Sprintf("Could not read s3 storage %s after update: %s", state.UUID.ValueString(), err))
		return
	}

	flattenS3Storage(s, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3StorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_s3_storage", "uuid": state.UUID.ValueString()})

	err := r.client.DeleteS3Storage(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting S3 storage", fmt.Sprintf("Could not delete s3 storage %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_s3_storage", "uuid": state.UUID.ValueString()})
}

func addCreateReadBackError(resp *resource.CreateResponse, uuid string, err error) {
	resp.Diagnostics.AddError(
		"S3 storage created but refresh failed",
		fmt.Sprintf(
			"Coolify created s3 storage %s, but the provider could not read it back: Could not read s3 storage %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.",
			uuid,
			uuid,
			err,
		),
	)
}

func (r *s3StorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// flattenS3Storage maps API fields into the Terraform model.
// Key and secret are preserved from state when the API omits them (sensitive).
func flattenS3Storage(s *client.S3Storage, m *s3StorageResourceModel) {
	m.UUID = types.StringValue(s.UUID)
	m.Name = types.StringValue(s.Name)
	if s.Description != "" || m.Description.IsNull() || m.Description.IsUnknown() {
		m.Description = flex.StringToFramework(s.Description)
	}
	m.Endpoint = types.StringValue(s.Endpoint)
	m.Bucket = types.StringValue(s.Bucket)
	m.Region = types.StringValue(s.Region)
	if s.Key != "" {
		m.Key = types.StringValue(s.Key)
	}
	if s.Secret != "" {
		m.Secret = types.StringValue(s.Secret)
	}
	if s.IsUsable != nil {
		m.IsUsable = types.BoolValue(*s.IsUsable)
	} else if m.IsUsable.IsNull() || m.IsUsable.IsUnknown() {
		m.IsUsable = types.BoolValue(false)
	}
}
