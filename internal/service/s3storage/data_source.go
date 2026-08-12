package s3storage

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*s3StorageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*s3StorageDataSource)(nil)
)

// s3StorageDataSource is the data source for a single Coolify S3 storage.
type s3StorageDataSource struct {
	client *client.Client
}

// s3StorageDataSourceModel maps the data source schema data.
// Key and secret are omitted: Coolify hides them unless the token can read sensitive data.
type s3StorageDataSourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`
}

// NewDataSource returns a new S3 storage data source instance.
func NewDataSource() datasource.DataSource {
	return &s3StorageDataSource{}
}

func (d *s3StorageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

func (d *s3StorageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a single Coolify S3 storage by its UUID. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the S3 storage to look up.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "A friendly name for the S3 storage.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the S3 storage.",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "S3-compatible endpoint URL.",
				Computed:            true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "S3 bucket name.",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "S3 region.",
				Computed:            true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify marks this storage as usable.",
				Computed:            true,
			},
		},
	}
}

func (d *s3StorageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *s3StorageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config s3StorageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_s3_storage"})

	uuid := config.UUID.ValueString()
	s, err := d.client.GetS3Storage(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading S3 storage", fmt.Sprintf("Could not read s3 storage %s: %s", uuid, err))
		return
	}

	config.UUID = types.StringValue(s.UUID)
	config.Name = types.StringValue(s.Name)
	config.Description = flex.StringToFramework(s.Description)
	config.Endpoint = types.StringValue(s.Endpoint)
	config.Bucket = types.StringValue(s.Bucket)
	config.Region = types.StringValue(s.Region)
	if s.IsUsable != nil {
		config.IsUsable = types.BoolValue(*s.IsUsable)
	} else {
		config.IsUsable = types.BoolValue(false)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
