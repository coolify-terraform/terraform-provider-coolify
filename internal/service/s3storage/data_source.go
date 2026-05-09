package s3storage

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &s3StorageDataSource{}
	_ datasource.DataSourceWithConfigure = &s3StorageDataSource{}
)

type s3StorageDataSource struct {
	client *client.Client
}

type s3StorageDataSourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
}

// NewDataSource returns a new S3 storage data source.
func NewDataSource() datasource.DataSource {
	return &s3StorageDataSource{}
}

func (d *s3StorageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

func (d *s3StorageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Coolify S3 storage destination.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the S3 storage.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the S3 storage.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the S3 storage.",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The S3 endpoint URL.",
				Computed:            true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "The S3 bucket name.",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The S3 region.",
				Computed:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "The S3 access key.",
				Computed:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "The S3 secret key.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *s3StorageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *s3StorageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config s3StorageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := d.client.GetS3Storage(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading S3 storage", err.Error())
		return
	}

	config.UUID = types.StringValue(s.UUID)
	config.Name = types.StringValue(s.Name)
	config.Description = types.StringValue(s.Description)
	config.Endpoint = types.StringValue(s.Endpoint)
	config.Bucket = types.StringValue(s.Bucket)
	config.Region = types.StringValue(s.Region)
	config.AccessKey = types.StringValue(s.AccessKey)
	config.SecretKey = types.StringValue(s.SecretKey)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
