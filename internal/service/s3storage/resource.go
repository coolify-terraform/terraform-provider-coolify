package s3storage

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Endpoint             types.String `tfsdk:"endpoint"`
	Bucket               types.String `tfsdk:"bucket"`
	Region               types.String `tfsdk:"region"`
	Key                  types.String `tfsdk:"key"`
	Secret               types.String `tfsdk:"secret"`
	UsePathStyleEndpoint types.Bool   `tfsdk:"use_path_style_endpoint"`
	IsUsable             types.Bool   `tfsdk:"is_usable"`
	Type                 types.String `tfsdk:"type"`
	SkipTLSVerification  types.Bool   `tfsdk:"skip_tls_verification"`
	PrivateKeyUUID       types.String `tfsdk:"private_key_uuid"`
	SFTPHostKey          types.String `tfsdk:"sftp_host_key"`
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
				MarkdownDescription: "Optional description of the S3 storage. Omitting the attribute later does not clear the stored Coolify value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "S3-compatible endpoint URL (e.g. `https://s3.us-east-1.amazonaws.com`). " +
					"Coolify validates this with its SafeWebhookUrl rule: private, loopback, and most internal hostnames " +
					"(including Docker DNS names such as `coolify-minio`) are rejected unless the Coolify instance " +
					"explicitly allowlists them. Use a public HTTPS endpoint for API-managed storages.",
				Required: true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "S3 bucket name. Required when type is `s3` or omitted. Do not set it for `sftp`, `ftps`, or `webdav`. Coolify rejects the field for those types.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "S3 region (for example `us-east-1`). Required when type is `s3` or omitted. Do not set it for `sftp`, `ftps`, or `webdav`. Coolify stores `us-east-1` as a placeholder for those types.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
			"use_path_style_endpoint": schema.BoolAttribute{
				MarkdownDescription: "Use path-style URLs (`endpoint/bucket`) instead of virtual-hosted-style (`bucket.endpoint`). Requires Coolify >= v4.5.1. On create, leaving it unset lets Coolify choose: false for Alibaba Cloud OSS, Tencent Cloud COS, and Volcengine TOS endpoints, and true for other endpoints. Removing the attribute later does not clear a stored value. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Storage type. `s3` (default when omitted), `sftp`, `ftps`, or `webdav`. A type other than `s3` requires Coolify >= v4.7.0. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.OneOf("s3", "sftp", "ftps", "webdav")},
			},
			"skip_tls_verification": schema.BoolAttribute{
				MarkdownDescription: "S3 only. Do not check the endpoint TLS certificate. Use only for a self-signed or private CA. Requires Coolify >= v4.7.0. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "SFTP only. UUID of a team private key used to log in. Requires Coolify >= v4.7.0. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{validate.UUID()},
			},
			"sftp_host_key": schema.StringAttribute{
				MarkdownDescription: "SFTP host key recorded by Coolify. Read-only.",
				Computed:            true,
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
	resp.Diagnostics.Append(rejectStoragePlan(r.client, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnUnsupportedPathStyle(r.client, plan.UsePathStyleEndpoint, &resp.Diagnostics)

	input := client.CreateS3StorageInput{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Endpoint:             plan.Endpoint.ValueString(),
		Bucket:               plan.Bucket.ValueString(),
		Region:               plan.Region.ValueString(),
		Key:                  plan.Key.ValueString(),
		Secret:               plan.Secret.ValueString(),
		UsePathStyleEndpoint: flex.BoolValueOrNull(plan.UsePathStyleEndpoint),
		IsUsable:             flex.BoolValueOrNull(plan.IsUsable),
		Type:                 plan.Type.ValueString(),
		SkipTLSVerification:  flex.BoolValueOrNull(plan.SkipTLSVerification),
		PrivateKeyUUID:       plan.PrivateKeyUUID.ValueString(),
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
		resp.Diagnostics.AddError(
			flex.CreateReadBackFailedSummary("S3 storage"),
			flex.CreateReadBackFailedDetail("s3 storage", createdUUID, err),
		)
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
	resp.Diagnostics.Append(rejectStoragePlan(r.client, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnUnsupportedPathStyle(r.client, plan.UsePathStyleEndpoint, &resp.Diagnostics)

	input := client.UpdateS3StorageInput{
		Name:                 flex.StringIfChanged(plan.Name, state.Name),
		Description:          flex.StringIfChanged(plan.Description, state.Description),
		Endpoint:             flex.StringIfChanged(plan.Endpoint, state.Endpoint),
		Bucket:               flex.StringIfChanged(plan.Bucket, state.Bucket),
		Region:               flex.StringIfChanged(plan.Region, state.Region),
		Key:                  flex.StringIfChanged(plan.Key, state.Key),
		Secret:               flex.StringIfChanged(plan.Secret, state.Secret),
		UsePathStyleEndpoint: flex.BoolIfChanged(plan.UsePathStyleEndpoint, state.UsePathStyleEndpoint),
		IsUsable:             flex.BoolIfChanged(plan.IsUsable, state.IsUsable),
		Type:                 flex.StringIfChanged(plan.Type, state.Type),
		SkipTLSVerification:  flex.BoolIfChanged(plan.SkipTLSVerification, state.SkipTLSVerification),
		PrivateKeyUUID:       flex.StringIfChanged(plan.PrivateKeyUUID, state.PrivateKeyUUID),
	}
	client.OmitDisallowedStorageUpdate(&input, plan.Type.ValueString())

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

func (r *s3StorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func warnUnsupportedPathStyle(c *client.Client, value types.Bool, diags *diag.Diagnostics) {
	if diags == nil || c == nil || c.SupportsCoolify451() {
		return
	}
	if value.IsNull() || value.IsUnknown() {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		"Coolify version cannot write use_path_style_endpoint",
		fmt.Sprintf(
			"This Coolify instance (%s) does not accept use_path_style_endpoint (added in Coolify v4.5.1). "+
				"The provider will keep the value in Terraform state but will not send it. "+
				"Upgrade to Coolify v4.5.1 or later, or remove use_path_style_endpoint from configuration.",
			ver,
		),
	)
}

// flattenS3Storage maps API fields into the Terraform model.
// Key and secret are preserved from state when the API omits them (sensitive).
func flattenS3Storage(s *client.S3Storage, m *s3StorageResourceModel) {
	m.UUID = types.StringValue(s.UUID)
	m.Name = types.StringValue(s.Name)
	flex.SetStringSeedOrClear(&m.Description, s.Description)
	m.Endpoint = types.StringValue(s.Endpoint)
	m.Bucket = types.StringValue(s.Bucket)
	m.Region = types.StringValue(s.Region)
	if s.Key != "" {
		m.Key = types.StringValue(s.Key)
	}
	if s.Secret != "" {
		m.Secret = types.StringValue(s.Secret)
	}
	if s.UsePathStyleEndpoint != nil {
		m.UsePathStyleEndpoint = types.BoolValue(*s.UsePathStyleEndpoint)
	} else if m.UsePathStyleEndpoint.IsUnknown() {
		m.UsePathStyleEndpoint = types.BoolNull()
	}
	if s.IsUsable != nil {
		m.IsUsable = types.BoolValue(*s.IsUsable)
	} else if m.IsUsable.IsNull() || m.IsUsable.IsUnknown() {
		m.IsUsable = types.BoolValue(false)
	}
	if s.Type != "" {
		m.Type = types.StringValue(s.Type)
	} else if m.Type.IsUnknown() {
		m.Type = types.StringNull()
	}
	if s.SkipTLSVerification != nil {
		m.SkipTLSVerification = types.BoolValue(*s.SkipTLSVerification)
	} else if m.SkipTLSVerification.IsUnknown() {
		m.SkipTLSVerification = types.BoolNull()
	}
	if s.PrivateKeyUUID != "" {
		m.PrivateKeyUUID = types.StringValue(s.PrivateKeyUUID)
	} else if m.PrivateKeyUUID.IsUnknown() {
		m.PrivateKeyUUID = types.StringNull()
	}
	if s.SFTPHostKey != "" {
		m.SFTPHostKey = types.StringValue(s.SFTPHostKey)
	} else if m.SFTPHostKey.IsNull() || m.SFTPHostKey.IsUnknown() {
		m.SFTPHostKey = types.StringNull()
	}
}
