package s3storage

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &s3StoragesDataSource{}
	_ datasource.DataSourceWithConfigure = &s3StoragesDataSource{}
)

type s3StoragesDataSource struct {
	client *client.Client
}

type s3StorageListItemModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
}

type s3StoragesDataSourceModel struct {
	Storages []s3StorageListItemModel `tfsdk:"storages"`
	Filters  []filter.Config          `tfsdk:"filter"`
}

// NewListDataSource returns a new data source that lists all S3 storages.
func NewListDataSource() datasource.DataSource {
	return &s3StoragesDataSource{}
}

func (d *s3StoragesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storages"
}

func (d *s3StoragesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all Coolify S3 storage destinations.\n\n" +
			"~> **Note:** Current versions of Coolify (v4) do not expose a public API for S3 storage CRUD. " +
			"This data source targets an API surface that may not be available in your Coolify version.",
		Attributes: map[string]schema.Attribute{
			"storages": schema.ListNestedAttribute{
				MarkdownDescription: "The list of S3 storages.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the S3 storage.",
							Computed:            true,
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
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *s3StoragesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *s3StoragesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config s3StoragesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	storages, err := d.client.ListS3Storages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing S3 storages", err.Error())
		return
	}

	storages = filter.Apply(storages, config.Filters, func(s client.S3Storage, field string) (string, bool) {
		switch field {
		case "uuid":
			return s.UUID, true
		case "name":
			return s.Name, true
		case "description":
			return s.Description, true
		case "endpoint":
			return s.Endpoint, true
		case "bucket":
			return s.Bucket, true
		case "region":
			return s.Region, true
		default:
			return "", false
		}
	})

	var state s3StoragesDataSourceModel
	state.Filters = config.Filters
	for _, s := range storages {
		state.Storages = append(state.Storages, s3StorageListItemModel{
			UUID:        types.StringValue(s.UUID),
			Name:        types.StringValue(s.Name),
			Description: types.StringValue(s.Description),
			Endpoint:    types.StringValue(s.Endpoint),
			Bucket:      types.StringValue(s.Bucket),
			Region:      types.StringValue(s.Region),
		})
	}

	if state.Storages == nil {
		state.Storages = []s3StorageListItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
