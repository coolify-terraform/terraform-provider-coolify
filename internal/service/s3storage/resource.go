package s3storage

import (
	"context"
	"fmt"
	"regexp"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &s3StorageResource{}
	_ resource.ResourceWithConfigure   = &s3StorageResource{}
	_ resource.ResourceWithImportState = &s3StorageResource{}
)

type s3StorageResource struct {
	client *client.Client
}

type s3StorageResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
}

// NewResource returns a new S3 storage resource.
func NewResource() resource.Resource {
	return &s3StorageResource{}
}

func (r *s3StorageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

func (r *s3StorageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify S3 storage destination.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the S3 storage.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the S3 storage.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the S3 storage.",
				Optional:            true,
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The S3 endpoint URL (must start with `http://` or `https://`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^https?://`), "must start with http:// or https://"),
				},
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "The S3 bucket name.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The S3 region.",
				Required:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "The S3 access key.",
				Required:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "The S3 secret key.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *s3StorageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *s3StorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan s3StorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateS3StorageInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Endpoint:    plan.Endpoint.ValueString(),
		Bucket:      plan.Bucket.ValueString(),
		Region:      plan.Region.ValueString(),
		AccessKey:   plan.AccessKey.ValueString(),
		SecretKey:   plan.SecretKey.ValueString(),
	}

	created, err := r.client.CreateS3Storage(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating S3 storage", err.Error())
		return
	}

	// Read back for full state.
	s, err := r.client.GetS3Storage(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading S3 storage after create", err.Error())
		return
	}

	mapS3StorageToModel(s, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3StorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := r.client.GetS3Storage(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading S3 storage", err.Error())
		return
	}

	mapS3StorageToModel(s, &state)
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

	name := plan.Name.ValueString()
	desc := plan.Description.ValueString()
	endpoint := plan.Endpoint.ValueString()
	bucket := plan.Bucket.ValueString()
	region := plan.Region.ValueString()
	accessKey := plan.AccessKey.ValueString()
	secretKey := plan.SecretKey.ValueString()

	input := client.UpdateS3StorageInput{
		Name:        &name,
		Description: &desc,
		Endpoint:    &endpoint,
		Bucket:      &bucket,
		Region:      &region,
		AccessKey:   &accessKey,
		SecretKey:   &secretKey,
	}

	_, err := r.client.UpdateS3Storage(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating S3 storage", err.Error())
		return
	}

	// Read back for full state.
	s, err := r.client.GetS3Storage(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading S3 storage after update", err.Error())
		return
	}

	mapS3StorageToModel(s, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3StorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteS3Storage(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting S3 storage", err.Error())
		return
	}
}

func (r *s3StorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func mapS3StorageToModel(s *client.S3Storage, model *s3StorageResourceModel) {
	model.UUID = types.StringValue(s.UUID)
	model.Name = types.StringValue(s.Name)
	model.Description = flex.StringToFramework(s.Description)
	model.Endpoint = types.StringValue(s.Endpoint)
	model.Bucket = types.StringValue(s.Bucket)
	model.Region = types.StringValue(s.Region)
	model.AccessKey = types.StringValue(s.AccessKey)
	model.SecretKey = types.StringValue(s.SecretKey)
}
